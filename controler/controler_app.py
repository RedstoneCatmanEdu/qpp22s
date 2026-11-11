import traceback
from os_ken.base import app_manager
from os_ken.controller import ofp_event
from os_ken.controller.handler import CONFIG_DISPATCHER, MAIN_DISPATCHER
from os_ken.controller.handler import set_ev_cls
from os_ken.ofproto import ofproto_v1_3
from os_ken.lib.packet import packet, ethernet, ipv4, udp
import parser as quic_parser
import os

class QuicLoadBalancer(app_manager.OSKenApp):
    OFP_VERSIONS = [ofproto_v1_3.OFP_VERSION]

    def __init__(self, *args, **kwargs):
        super(QuicLoadBalancer, self).__init__(*args, **kwargs)
        self.logger.info("QUIC Load Balancer Started")
        
        self.vip_ip = '192.168.10.100'
        self.vip_mac = '00:00:00:00:00:aa'
        self.client_ip = '192.168.10.10'
        self.client_port = 1 
        
        self.servers = {
            1: {'ip': '10.0.0.11', 'mac': '00:00:00:00:00:11', 'port': 2},
            2: {'ip': '10.0.0.12', 'mac': '00:00:00:00:00:12', 'port': 3}
        }
        self.next_server_id = 1 
        self.connection_map = {}
        self.datapath = None
        
        self.flow_idle_timeout = int(os.environ.get('FLOW_IDLE_TIMEOUT', 60))
        self.logger.info(f"OpenFlow rule idle timeout set to: {self.flow_idle_timeout} sec")

    @set_ev_cls(ofp_event.EventOFPSwitchFeatures, CONFIG_DISPATCHER)
    def switch_features_handler(self, ev):
        self.logger.info("=== SWITCH CONNECTED ===")
        self.datapath = ev.msg.datapath
        ofproto = self.datapath.ofproto
        parser = self.datapath.ofproto_parser
        
        match = parser.OFPMatch()
        actions = [parser.OFPActionOutput(ofproto.OFPP_CONTROLLER, ofproto.OFPCML_NO_BUFFER)]
        self.add_flow(self.datapath, 0, match, actions)
        
        match_arp = parser.OFPMatch(eth_type=0x0806)
        actions_arp = [parser.OFPActionOutput(ofproto.OFPP_FLOOD)]
        self.add_flow(self.datapath, 1, match_arp, actions_arp)
        
        self.logger.info("Default Packet-In and ARP flood rules installed")

    def add_flow(self, datapath, priority, match, actions):
        ofproto = datapath.ofproto
        parser = datapath.ofproto_parser
        inst = [parser.OFPInstructionActions(ofproto.OFPIT_APPLY_ACTIONS, actions)]
        
        mod = parser.OFPFlowMod(
            datapath=datapath, 
            priority=priority,
            idle_timeout=self.flow_idle_timeout, 
            match=match, 
            instructions=inst
        )
        datapath.send_msg(mod)

    @set_ev_cls(ofp_event.EventOFPPacketIn, MAIN_DISPATCHER)
    def _packet_in_handler(self, ev):
        try:
            if self.datapath is None:
                return

            msg = ev.msg
            datapath = self.datapath
            ofproto = datapath.ofproto
            parser = datapath.ofproto_parser
            
            pkt = packet.Packet(msg.data)
            eth = pkt.get_protocol(ethernet.ethernet)
            
            if eth.ethertype != 0x0800:
                actions = [parser.OFPActionOutput(ofproto.OFPP_FLOOD)]
                out = parser.OFPPacketOut(datapath=datapath, buffer_id=ofproto.OFP_NO_BUFFER, in_port=msg.match['in_port'], actions=actions, data=msg.data)
                datapath.send_msg(out)
                return

            ip_pkt = pkt.get_protocol(ipv4.ipv4)
            udp_pkt = pkt.get_protocol(udp.udp)

            if not udp_pkt or udp_pkt.dst_port != 443 or ip_pkt.dst != self.vip_ip:
                actions = [parser.OFPActionOutput(ofproto.OFPP_FLOOD)]
                out = parser.OFPPacketOut(datapath=datapath, buffer_id=ofproto.OFP_NO_BUFFER, in_port=msg.match['in_port'], actions=actions, data=msg.data)
                datapath.send_msg(out)
                return

            self.logger.info(f"Packet-In: UDP {ip_pkt.src}:{udp_pkt.src_port} -> VIP:443")
            
            quic_bytes = None
            protocols = pkt.protocols
            if udp_pkt in protocols:
                idx = protocols.index(udp_pkt)
                if idx + 1 < len(protocols):
                    quic_bytes = protocols[idx + 1]
            
            if not quic_bytes or not isinstance(quic_bytes, bytes):
                return

            parsed = quic_parser.parse_quic_header(quic_bytes)
            flow_key = (ip_pkt.src, udp_pkt.src_port)
            
            server_id = None
            if parsed:
                dcid = parsed['dcid']
                server_id = quic_parser.decode_backend_id(dcid)
                self.logger.info(f"Parsed DCID: {dcid.hex()}, Server ID: {server_id}")

            if server_id is not None:
                self.logger.info(f"Routed by Server CID. Server ID: {server_id}")
                self.connection_map[flow_key] = server_id
            else:
                if parsed and parsed.get('packet_type') == 'initial':
                    if flow_key in self.connection_map:
                        server_id = self.connection_map[flow_key]
                        self.logger.info(f"Retransmitted Initial for flow {flow_key}. Reusing Server {server_id}")
                    else:
                        server_id = self.next_server_id
                        self.connection_map[flow_key] = server_id
                        self.next_server_id = 2 if self.next_server_id == 1 else 1
                        self.logger.info(f"New connection for flow {flow_key}! Round Robin chose Server {server_id}")
                else:
                    self.logger.info("Not a valid QUIC packet")
                    return

            if server_id not in self.servers:
                return
            server = self.servers[server_id]
            
    
            client_ip = ip_pkt.src
            client_port_ofp = msg.match['in_port']

            match_in = parser.OFPMatch(
                in_port=client_port_ofp, eth_type=0x0800, ip_proto=17,
                ipv4_src=client_ip, ipv4_dst=self.vip_ip,
                udp_src=udp_pkt.src_port, udp_dst=443
            )
            actions_in = [
                parser.OFPActionSetField(eth_dst=server['mac']),
                parser.OFPActionSetField(ipv4_dst=server['ip']),
                parser.OFPActionOutput(server['port'])
            ]
            self.add_flow(datapath, 10, match_in, actions_in)

            match_out = parser.OFPMatch(
                in_port=server['port'], eth_type=0x0800, ip_proto=17,
                ipv4_src=server['ip'], ipv4_dst=client_ip, 
                udp_src=443, udp_dst=udp_pkt.src_port
            )
            actions_out = [
                parser.OFPActionSetField(eth_src=self.vip_mac),
                parser.OFPActionSetField(ipv4_src=self.vip_ip),
                parser.OFPActionOutput(client_port_ofp) 
            ]
            self.add_flow(datapath, 10, match_out, actions_out)

            actions_send = [
                parser.OFPActionSetField(eth_dst=server['mac']),
                parser.OFPActionSetField(ipv4_dst=server['ip']),
                parser.OFPActionOutput(server['port'])
            ]
            out = parser.OFPPacketOut(
                datapath=datapath, buffer_id=ofproto.OFP_NO_BUFFER, 
                in_port=client_port_ofp, actions=actions_send, data=msg.data
            )
            datapath.send_msg(out)

            self.logger.info(f"Installed flows and forwarded packet to Server {server_id}")
            
        except Exception as e:
            self.logger.error(f"ERROR: {e}")
            self.logger.error(traceback.format_exc())
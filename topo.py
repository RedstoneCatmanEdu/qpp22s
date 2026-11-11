from mininet.topo import Topo
from mininet.net import Mininet
from mininet.node import RemoteController, OVSKernelSwitch
from mininet.cli import CLI
from mininet.log import setLogLevel, info
import os
import argparse

class QuicTopo(Topo):
    def build(self):
        s1 = self.addSwitch('s1')
        client = self.addHost('client', ip='192.168.10.10/24', mac='00:00:00:00:00:01')
        
        server1 = self.addHost('server1', ip='10.0.0.11/24', mac='00:00:00:00:00:11')
        server2 = self.addHost('server2', ip='10.0.0.12/24', mac='00:00:00:00:00:12')

        self.addLink(client, s1)
        self.addLink(server1, s1)
        self.addLink(server2, s1)

def run(auto_start=False):
    setLogLevel('info')
    topo = QuicTopo()
    
    net = Mininet(topo=topo, switch=OVSKernelSwitch, controller=None, autoSetMacs=False)
    net.addController('c0', controller=RemoteController, ip='127.0.0.1', port=6653)
    
    net.start()
    
    info('*** Configuring VIP on switch s1 ***\n')
    net.get('s1').cmd('ifconfig s1 192.168.10.100/24 up')
    
    info('*** Configuring static ARP on client ***\n')
    net.get('client').cmd('arp -s 192.168.10.100 00:00:00:00:00:aa')
    
    info('*** Adding routes ***\n')
    net.get('server1').cmd('ip route add 192.168.10.0/24 dev server1-eth0')
    net.get('server2').cmd('ip route add 192.168.10.0/24 dev server2-eth0')
    net.get('client').cmd('ip route add 10.0.0.0/24 dev client-eth0')
    
    project_root = os.environ.get('PROJECT_ROOT', '.')
    
    if auto_start:
        info('*** Starting QUIC Servers automatically ***\n')
        net.get('server1').cmd(f'{project_root}/bin/server -id 1 > /tmp/server1.log 2>&1 &')
        net.get('server2').cmd(f'{project_root}/bin/server -id 2 > /tmp/server2.log 2>&1 &')
    else:
        info('*** Servers NOT started automatically (use -a to auto-start) ***\n')
        info('*** To start them manually inside Mininet, run: ***\n')
        info(f'*** server1 {project_root}/bin/server -id 1 & ***\n')
        info(f'*** server2 {project_root}/bin/server -id 2 & ***\n')
    
    info('*** Network is up. Entering Mininet CLI... ***\n')
    info('*** Example: client ./bin/client -addr 192.168.10.100 -port 443 -c 4 ***\n')
    
    CLI(net)
    net.stop()

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description="Mininet Topology for QUIC Load Balancer")
    parser.add_argument('-a', '--auto', action='store_true', help="Auto-start QUIC servers (server1 and server2)")
    args = parser.parse_args()
    run(auto_start=args.auto)
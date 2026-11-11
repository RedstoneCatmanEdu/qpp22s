import struct

CONFIG_ID = 0xA1
CID_LENGTH = 12
QUIC_V1 = 0x00000001

def decode_backend_id(dcid: bytes) -> int | None:
    if len(dcid) != CID_LENGTH:
        return None
    if dcid[0] != CONFIG_ID:
        return None
    server_id = dcid[1]
    if server_id in (1, 2):
        return server_id
    return None

def parse_long_header(payload: bytes) -> dict | None:
    if len(payload) < 7:
        return None
        
    first_byte = payload[0]
    
    packet_type_bits = (first_byte & 0x30) >> 4
    packet_types = {0: "initial", 1: "0-RTT", 2: "handshake", 3: "retry"}
    packet_type = packet_types.get(packet_type_bits, "unknown")

    version = struct.unpack("!I", payload[1:5])[0] 
    if version != QUIC_V1:
        return None
        
    dcid_len = payload[5]
    if len(payload) < 6 + dcid_len + 1:
        return None 
        
    dcid = payload[6 : 6+dcid_len]
    
    scid_len_offset = 6 + dcid_len
    scid_len = payload[scid_len_offset]
    
    if len(payload) < scid_len_offset + 1 + scid_len:
        return None
        
    scid = payload[scid_len_offset+1 : scid_len_offset+1+scid_len]
    
    return {
        "header_type": "long",
        "packet_type": packet_type,
        "version": version,
        "dcid": dcid,
        "scid": scid
    }

def parse_short_header(payload: bytes) -> dict | None:
    if len(payload) < 1 + CID_LENGTH:
        return None
        
    dcid = payload[1 : 1 + CID_LENGTH]
    
    return {
        "header_type": "short",
        "dcid": dcid
    }

def parse_quic_header(payload: bytes) -> dict | None:
    if len(payload) < 1:
        return None
        
    first_byte = payload[0]

    header_form = (first_byte & 0x80) >> 7
    
    if header_form == 1:
        return parse_long_header(payload)
    else:
        return parse_short_header(payload)
    
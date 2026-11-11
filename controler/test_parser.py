import pytest
import struct
import os
import parser as quic_parser

# ==========================================
# Helper functions to generate packets
# ==========================================

def create_long_header_packet(dcid_hex: str, scid_hex: str, version: int = 1) -> bytes:
    """Creates a valid Long Header packet (Initial)."""
    first_byte = 0xC0 # 1 (Long Header) + 1 (Fixed Bit) + 00 (Initial)
    ver_bytes = struct.pack("!I", version)
    dcid = bytes.fromhex(dcid_hex)
    scid = bytes.fromhex(scid_hex)
    
    packet = (
        bytes([first_byte]) + 
        ver_bytes + 
        bytes([len(dcid)]) + dcid + 
        bytes([len(scid)]) + scid + 
        b"dummy_payload_data"
    )
    return packet

def create_short_header_packet(dcid_hex: str) -> bytes:
    """Creates a valid Short Header packet (1-RTT)."""
    first_byte = 0x40 # 0 (Short Header) + 1 (Fixed Bit)
    dcid = bytes.fromhex(dcid_hex)
    return bytes([first_byte]) + dcid + b"dummy_payload_data"

# ==========================================
# 1. Tests for parse_quic_header (Long/Short)
# ==========================================

def test_parse_valid_long_header():
    packet = create_long_header_packet("a1014b829157304c71a8b92d", "1122334455667788")
    result = quic_parser.parse_quic_header(packet)
    
    assert result is not None
    assert result["header_type"] == "long"
    assert result["version"] == 1
    assert result["packet_type"] == "initial"
    assert result["dcid"].hex() == "a1014b829157304c71a8b92d"
    assert result["scid"].hex() == "1122334455667788"

def test_parse_valid_short_header():
    packet = create_short_header_packet("a1014b829157304c71a8b92d")
    result = quic_parser.parse_quic_header(packet)
    
    assert result is not None
    assert result["header_type"] == "short"
    assert result["dcid"].hex() == "a1014b829157304c71a8b92d"

def test_parse_unsupported_version():
    # Version 2 (unsupported)
    packet = create_long_header_packet("a1014b829157304c71a8b92d", "1122", version=2)
    result = quic_parser.parse_quic_header(packet)
    assert result is None

def test_parse_truncated_long_header():
    # Packet shorter than the minimum 7 bytes
    packet = b"\xc0\x00\x00"
    result = quic_parser.parse_quic_header(packet)
    assert result is None

def test_parse_truncated_short_header():
    # Packet shorter than 13 bytes (1 flag + 12 DCID)
    packet = b"\x40\xa1\x01"
    result = quic_parser.parse_quic_header(packet)
    assert result is None

def test_parse_empty_payload():
    result = quic_parser.parse_quic_header(b"")
    assert result is None

def test_parse_random_garbage():
    # Feed 100 random packets. The parser must not crash.
    for i in range(100):
        random_len = i % 50
        random_packet = os.urandom(random_len)
        quic_parser.parse_quic_header(random_packet) # Should not raise exceptions

# ==========================================
# 2. Tests for decode_backend_id
# ==========================================

@pytest.mark.parametrize("cid_hex, expected_server_id", [
    ("a1014b829157304c71a8b92d", 1), # Valid CID for Server 1
    ("a1029c1742f5186394ab7721", 2), # Valid CID for Server 2
])
def test_decode_valid_cid(cid_hex, expected_server_id):
    cid = bytes.fromhex(cid_hex)
    result = quic_parser.decode_backend_id(cid)
    assert result == expected_server_id

def test_decode_invalid_length():
    # 11 bytes instead of 12
    cid = bytes.fromhex("a1014b829157304c71a8b9")
    assert quic_parser.decode_backend_id(cid) is None

def test_decode_wrong_configuration_id():
    # First byte is 0xB1 instead of 0xA1
    cid = bytes.fromhex("b1014b829157304c71a8b92d")
    assert quic_parser.decode_backend_id(cid) is None

def test_decode_unknown_server_id():
    # Server ID = 9 (we only have 1 and 2)
    cid = bytes.fromhex("a1094b829157304c71a8b92d")
    assert quic_parser.decode_backend_id(cid) is None

def test_decode_empty_cid():
    assert quic_parser.decode_backend_id(b"") is None

# ==========================================
# 3. Integration Test (Packet -> Decoder)
# ==========================================

def test_full_pipeline_long_header():
    """Tests the full chain: from raw bytes to the server ID."""
    packet = create_long_header_packet("a1024b829157304c71a8b92d", "1122")
    parsed = quic_parser.parse_quic_header(packet)
    
    assert parsed is not None
    server_id = quic_parser.decode_backend_id(parsed["dcid"])
    assert server_id == 2


def test_decode_multiple_different_cids_for_server1():
    """
    Tests that different CIDs from the same server are routed to the same backend.
    Using exact examples from the assignment PDF.
    """
    original_cid = bytes.fromhex("a1014b829157304c71a8b92d")
    new_cid = bytes.fromhex("a1019811aa75e3f0a221d47b")
    
    # Both must return Server 1
    assert quic_parser.decode_backend_id(original_cid) == 1
    assert quic_parser.decode_backend_id(new_cid) == 1

def test_multiple_cids_same_server():
    cid1 = bytes.fromhex("a1014b829157304c71a8b92d")
    cid2 = bytes.fromhex("a1019811aa75e3f0a221d47b")
    assert quic_parser.decode_backend_id(cid1) == 1
    assert quic_parser.decode_backend_id(cid2) == 1
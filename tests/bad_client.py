import socket
import time

s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
vip = ('192.168.10.100', 443)

print('Sending random non-QUIC UDP packet...')
s.sendto(b'THIS_IS_JUST_RANDOM_GARBAGE', vip)

print('Sending truncated QUIC header...')
s.sendto(b'\xc0\x00', vip)

print('Sending invalid CID (wrong config ID)...')
s.sendto(b'\x40\xb1\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c', vip)

print('Sending invalid Server ID (9)...')
s.sendto(b'\x40\xa1\x09\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c', vip)

time.sleep(1)
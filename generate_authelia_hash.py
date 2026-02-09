import hashlib
import os
import base64
import sys

if len(sys.argv) < 2:
    print("Usage: python3 generate_authelia_hash.py <password>")
    sys.exit(1)

password = sys.argv[1]
salt = os.urandom(16)
iterations = 310000
digest = 'sha512'

dk = hashlib.pbkdf2_hmac(digest, password.encode(), salt, iterations)

# Authelia uses RFC4648 without padding
b64_salt = base64.b64encode(salt).decode().rstrip('=')
b64_hash = base64.b64encode(dk).decode().rstrip('=')

full_hash = f'$pbkdf2-{digest}${iterations}${b64_salt}${b64_hash}'
print(f"Length: {len(full_hash)}")
for i in range(0, len(full_hash), 50):
    print(f"Chunk {i//50}: {full_hash[i:i+50]}")

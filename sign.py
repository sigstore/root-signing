from tuf.repository_tool import *
import argparse
import os
import pathlib
import shutil
import tempfile
import subprocess
from securesystemslib.keys import create_signature
from securesystemslib.formats import encode_canonical
from securesystemslib.signer import Signature
from securesystemslib import util as sslib_util
import base64


'''
Creates and appends a signature using cosign and your HSM.
Only run this after all (TOTAL_KEYS) have been added to the metadata files!

Usage:
python3 sign.py <SERIAL_NUMBER>

Args:
  SERIAL_NUMBER: The 8 digit serial number associated with your Yubikey.

TODO: Investiagate whether we can cache pins
'''

YUBIKEY_DIRECTORY = 'ceremony/2021-05-03/ceremony-products'
# Total number of keys we expect in the metadata before signing.
TOTAL_KEYS = 5

def get_key(serial_number):
    # TODO: Handle errors
    pem_file = open(os.path.join(YUBIKEY_DIRECTORY, serial_number, serial_number + "_pubkey.pem"), 'r')
    pem = pem_file.read()
    pubkey = import_ecdsakey_from_pem(pem)
    pubkey['keytype'] = 'ecdsa-sha2-nistp256'
    return pubkey

def create_signature(content, key):
    tmp = tempfile.NamedTemporaryFile('w+t')
    tmp.write(content)
    tmp.flush()
    ps = subprocess.run(["cosign", "sign-blob", "-sk", tmp.name], text=True, stdout=subprocess.PIPE)
    sig = base64.b64decode(ps.stdout.strip()).hex()
    tmp.close()
    return {'keyid': key['keyid'], 'method': 'ecdsa-sha2-nistp256', 'sig': sig}


def add_signature(key, path):
    # TODO: Check if signature already exists.
    content = dump_signable_metadata(path)
    append_signature(create_signature(content, key), path)
    return


def main():
    parser = argparse.ArgumentParser(description='Create and generate a signature for TUF metadata.')
    parser.add_argument('serial_number', help='your keys serial number')
    args = parser.parse_args()
    key = get_key(args.serial_number) 

    # Sanity check that there are the right number of keys added before signing!
    signable = sslib_util.load_json_file(os.path.join('repository','metadata.staged','root.json'))
    assert len(signable['signed']['keys']) == TOTAL_KEYS, "Attempting to sign metadata before all keys have been added!"

    # TODO: Check if you have already signed to avoid doubling up on key_ids.

    # Snapshot and timestamp are here to avoid complaints by the tool.
    roles = ['root', 'targets', 'snapshot', 'timestamp']
    for role in roles:
        print("Creating signature for %s" % role)
        add_signature(key, os.path.join('repository','metadata.staged', role + '.json'))

    # If we have the right number of keys, move this to a metadata/ directory.

if __name__ == "__main__":
    main()

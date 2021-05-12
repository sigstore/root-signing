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
import base64


'''
Creates and appends a signature using cosign and your HSM.
'''

YUBIKEY_DIRECTORY = 'ceremony/2021-05-03/ceremony-products'

def get_key(serial_number):
    # TODO: Handle errors
    pem_file = open(os.path.join(YUBIKEY_DIRECTORY, serial_number, serial_number + "_pubkey.pem"), 'r')
    pem = pem_file.read()
    pubkey = import_ecdsakey_from_pem(pem)
    return pubkey

def add_signature(key, path):
    tmp = tempfile.NamedTemporaryFile('w+t')
    tmp.write(str(encode_canonical(dump_signable_metadata(path)).encode()))
    p = subprocess.run(["cosign", "sign-blob", "-sk", str(tmp.name)], text=True, stdout=subprocess.PIPE)
    sig = base64.b64decode(p.stdout.rstrip()).hex()
    signature = Signature(key['keyid'], sig)
    append_signature({'keyid': key['keyid'], 'method': 'ecdsa-sha2-nistp256', 'sig': sig}, path)
    tmp.close()
    return


def main():
    parser = argparse.ArgumentParser(description='Create and generate a signature for TUF metadata.')
    parser.add_argument('serial_number', help='your keys serial number')
    args = parser.parse_args()
    key = get_key(args.serial_number) 
    for path in ['root.json', 'targets.json']:
        print("Creating signature for %s" % path)
        add_signature(key, os.path.join('repository','metadata.staged',path))


if __name__ == "__main__":
    main()

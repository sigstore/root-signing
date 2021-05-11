from tuf.repository_tool import *
import argparse
import os
import pathlib
import shutil
from securesystemslib.keys import create_signature
from securesystemslib.formats import encode_canonical


'''
Creates and appends a signature to the root and target metadata.
'''

def add_signature(private_key, path):
    content = dump_signable_metadata(path)
    signature = create_signature(private_key, encode_canonical(content).encode())
    append_signature(signature, path)


def main():
    parser = argparse.ArgumentParser(description='Create and generate a signature for TUF metadata.')
    parser.add_argument('private_key', type=argparse.FileType('r'))
    args = parser.parse_args()
    private_key = import_ecdsakey_from_pem(args.private_key.read())
    print(private_key)
    for path in ['root.json', 'targets.json']:
        add_signature(private_key, os.path.join('repository','metadata.staged',path))


if __name__ == "__main__":
    main()

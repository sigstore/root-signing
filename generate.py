from tuf.repository_tool import *
import os
import collections
import datetime
import pathlib
import shutil

'''
Generates unsigned metadata from a collection of TUF root keys from YUBIKEY_DIRECTORY, 
target files from the TARGETS_DIR and a key THRESHOLD.

Usage:
python3 generate.py

Root, target, snapshot, and timestamp metadata files will be written to 
repository/metadata.staged.

To avoid creating multiple versions, this rewrites repository metadata each time
it is run.
'''

YUBIKEY_DIRECTORY = 'ceremony/2021-05-03/ceremony-products'
# TODO: Add more targets to thie directory.
TARGETS_DIR = 'targets/'
THRESHOLD = 3


# Import Yubikey files. ecdsa keys
def get_yubikeys(products):
    ''' Get a list of public keys '''
    keys = []
    for root, dirs, files in os.walk(products):
        for dir_name in dirs:
            d = os.path.join(products, dir_name)
            for f in os.listdir(d):
                filename = os.path.join(d, f)
                if filename.endswith('pubkey.pem'):
                    pem_file = open(filename, 'r')
                    pem = pem_file.read()
                    pubkey = import_ecdsakey_from_pem(pem)
                    pubkey['keytype'] = 'ecdsa-sha2-nistp256'
                elif filename.endswith('device_cert.pem'):
                    dir_hardware_cert = filename
                elif filename.endswith('key_cert.pem'):
                    dir_cert = filename
            keys.append(pubkey)
    return keys


def get_targets(targets_dir):
    return [x[2] for x in os.walk(targets_dir)][0]


def main():
    print("Creating new repository...")
    repository = create_new_repository("repository")

    yubikeys = get_yubikeys(YUBIKEY_DIRECTORY)

    # Metadata expiration
    role_delta = datetime.timedelta(weeks=16)
    role_date = datetime.date.today() + role_delta
    role_expiration = datetime.datetime(role_date.year, role_date.month, role_date.day)
    # Key expiration.
    # TODO: This expiration and the attestation info does not show up in the metadata
    # ssl KEY_SCHEMA. Should we add it here?
    key_delta = datetime.timedelta(weeks=52)
    key_date = datetime.date.today() + key_delta
    key_expiration = datetime.datetime(key_date.year, key_date.month, key_date.day)
    # Including the snapshot and timestamp because if they don't exist, the repo tool complains.
    for role in [repository.root, repository.targets, repository.snapshot, repository.timestamp]:
        role.threshold = THRESHOLD
        role.expiration = role_expiration
        for key in yubikeys:
            role.add_verification_key(key, expires=key_expiration)

    # Add target files.
    targets = get_targets(TARGETS_DIR)
    repository.targets.add_targets(targets)
    for t in targets:
        target_name = pathlib.Path(t).name
        target = os.path.join(TARGETS_DIR, t)
        print(target)
        shutil.copy2(target, os.path.join('repository/targets', target_name))

    roles = ['root', 'targets', 'snapshot', 'timestamp']
    repository.mark_dirty(roles)
    for role in roles:
        repository.write(rolename = role)

if __name__ == "__main__":
    main()

# test-sigstore-root

This repository contains the steps and programs needed to create signed TUF metadata for SigStore. 

## Start

0. Install TUF.
```
pip3 install --user tuf
```

Make sure cosign is on your system path.

1. Provision and add your keys with
```
python3 provision.py
```
Take note of your serial number.

2. Wait until everyone has provisioned their keys. Generate should only be run after all keys are provisioned. This is a TODO right now.

3. Run 
```
python3 sign.py <SERIAL_NUMBER>
```




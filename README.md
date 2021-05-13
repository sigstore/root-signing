# test-sigstore-root

This repository contains the steps and programs needed to create signed TUF metadata for SigStore. 

## Start

0. Install TUF.
```
pip3 install --user tuf
```

Make sure cosign is on your system path.

1. Each keyholder should provision and add their keys with
```
python3 provision.py
```
Take note of your serial number.

2. When everyone has completed generating their keys, run
```
python3 generate.py
```

3. Run 
```
python3 sign.py <SERIAL_NUMBER>
```




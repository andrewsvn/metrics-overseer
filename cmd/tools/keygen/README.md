# cmd/tools/keygen

__keygen__ tool is a simple RSA key pair generator which creates RSA keys in files 
to support data encryption/decryption between metrics-overseer server and agent.

## File format
keygen generates both private and public keys in a separate PEM files with content 
in PKCS1 format. Files can be then processed using pem package.

## Usage
keygen tool creates two files under one given base directory, but each file can have its
own relative subpath - so they might be stored in different directories as a result.

__Basic usage__:

``` shell
keygen <base_directory> -pr=<path_to_private_key_file> -pb=<path_to_public_key_file> 
-bits=<rsa_key_bits>
```

Each parameter can be omitted with the following default values:
- base_directory = "."
- path_to_private_key_file = "private.pem"
- path_to_public_key_file = "public.pem"
- rsa_key_bits = 2048

__Note__ that RSA standard doesn't allow to use keys lesser than 1024 bits 
(2048 and above recommended).
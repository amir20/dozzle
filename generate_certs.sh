openssl genpkey -algorithm RSA -out shared_key.pem -pkeyopt rsa_keygen_bits:2048
openssl req -new -key shared_key.pem -out shared_request.csr -subj "/C=US/ST=California/L=San Francisco/O=Dozzle"
openssl x509 -req -in shared_request.csr -signkey shared_key.pem -out shared_cert.pem -days 365

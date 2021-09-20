# KubPoc
POC to demonstrated how to provision a simple Amazon EKS with Pulumi
 
A simple Nginx server is deployed on the Kubernetes instance
[![asciicast](https://asciinema.org/a/436807.svg)](https://asciinema.org/a/436807?t=5&speed=4&autoplay=1)
 
## Links
* Demo site http://kubpoc.jmob.net, curently stoped for cost optimization :-)
* Pulumi stack https://app.pulumi.com/yyvess/KubPoc/dev (private)
* Pulumi doc https://www.pulumi.com/
* Pulumi EKS Guide https://www.pulumi.com/docs/guides/crosswalk/aws/eks/
* Mozilla ssl generator https://ssl-config.mozilla.org/
* Ssl analyse https://www.ssllabs.com/ssltest/analyze.html?d=kubpoc.jmob.net&hideResults=on&latest
## Nginx
Nginx access is restricted by ssl client certificate, see [nginx.conf](./app/nginx.conf)
## Certificates
Certificates are signed with Let's encrypt was generated with certbot
## Client certificates
To access Nginx a client certificate is required. 

[testuser.pfx](./client/testuser.pfx) can be import to access Nginx site, the certificate password is 'test'
### Client certificate generation
```
# Generate the CA
openssl genrsa -des3 -out testuser.key 2048
openssl req -new -key testuser.key -out testuser.csr
openssl x509 -in myca.crt -out myca.pem -outform PEM
# Generate a client certificate
openssl genrsa -des3 -out testuser.key 2048
openssl req -new -key testuser.key -out testuser.csr
# Sign with our certificate-signing CA
openssl x509 -req -days 365 -in testuser.csr -CA myca.crt -CAkey myca.key -set_serial 01 -out testuser.crt
# Combined the key material into a single PFX.
openssl pkcs12 -export -out testuser.pfx -inkey testuser.key -in testuser.crt -certfile myca.crt
```
### Deploy the stack
```
pulumi up
```
### Setup kubctrl access
```
pulumi stack output kubeconfig > kubeconfig.yml
export KUBECONFIG=./kubeconfig.yml
kubectl get nodes
```

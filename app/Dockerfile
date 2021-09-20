FROM nginx:1.21-alpine
COPY ./nginx.conf /etc/nginx/nginx.conf
COPY pki /pki
RUN chown -R 0:0 /pki && \
  chmod 0555 /pki && \
  chmod 0222 /pki/* && \
  chown -R 0:0 /etc/nginx && \
  chmod 0222 /etc/nginx/* 

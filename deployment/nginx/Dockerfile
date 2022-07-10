FROM nginx

RUN mkdir -p /usr/share/nginx/html/vue/axis-chat
RUN mkdir -p /usr/share/upload/static
RUN mkdir -p /etc/nginx/cert
COPY ./cert/ /etc/nginx/cert/
COPY ./frontend/axis-chat/ /usr/share/nginx/html/vue/axis-chat/
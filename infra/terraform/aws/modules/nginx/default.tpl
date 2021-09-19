%{ for route in routes ~}
upstream ${ route.name } {
%{ for server in route.servers ~}
    server ${ server };
%{ endfor ~}
}
%{ endfor ~}

server {
	listen 80 default_server;
	listen [::]:80 default_server;


%{ for route in routes ~}
	location /${ route.name } {
#		 rewrite /${ route.name }/(.*) /$1;
		 proxy_pass http://${ route.name };
	}
%{ endfor ~}

	location / {
		 proxy_pass http://127.0.0.1:8000;
	}
}

server {
	listen 8000 default_server;
	listen [::]:8000 default_server;

	root /var/www/html;

	index index.html;

	server_name _;

	location / {
		try_files $uri $uri/ =404;
	}
}

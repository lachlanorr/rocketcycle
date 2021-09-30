%{ for route in routes ~}
upstream ${ route.name } {
%{ for server in [for host in route.hosts: "${host}:${route.port}"] ~}
	server ${ server };
%{ endfor ~}
}
%{ endfor ~}

server {
	listen 80;
	listen [::]:80;


%{ for route in routes ~}
%{ if !lookup(route, "grpc", false) ~}
	location /${ route.name } {
#		 rewrite /${ route.name }/(.*) /$1;
		 proxy_pass http://${ route.name };
	}
%{ endif ~}
%{ endfor ~}

	location / {
		 proxy_pass http://127.0.0.1:8000;
	}
}

server {
	listen 8000;
	listen [::]:8000;

	root /var/www/html;

	index index.html;

	server_name _;

	location / {
		try_files $uri $uri/ =404;
	}
}
%{ for route in routes ~}
%{ if lookup(route, "grpc", false) ~}

server {
	listen ${ route.port } http2;
	listen [::]:${ route.port } http2;

	location / {
		grpc_pass grpc://${ route.name };
	}
}
%{ endif ~}
%{ endfor ~}

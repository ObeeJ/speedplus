FROM ghcr.io/project-osrm/osrm-backend:v5.27.1

RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

COPY start.sh /start.sh
RUN chmod +x /start.sh

EXPOSE 5000

ENTRYPOINT ["/start.sh"]

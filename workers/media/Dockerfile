FROM python:3.12-slim-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg && rm -rf /var/lib/apt/lists/*
WORKDIR /worker
COPY server.py .
USER 65534:65534
ENV BIND_HOST=0.0.0.0 PORT=8090
EXPOSE 8090
CMD ["python", "server.py"]

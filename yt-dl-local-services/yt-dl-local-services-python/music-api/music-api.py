import http.server
import socketserver
import threading
import time
import yaml
import os
import json

from ytmusicapi import YTMusic
from urllib.parse import urlparse, parse_qs

ytmusic = YTMusic()

shutdown_flag = threading.Event()

def load_config(config_path):
    if not os.path.exists(config_path):
        raise FileNotFoundError(f"Configuration file not found: {config_path}")

    try:
        with open(config_path, 'r') as f:
            config = yaml.safe_load(f)
    except yaml.YAMLError as e:
        raise RuntimeError(f"Error parsing YAML configuration: {e}")

    if 'local_port_python_services' not in config:
        raise KeyError("Missing required configuration key: 'local_port_python_services'")

    return config

# Load the configuration
config_path = os.path.join(os.path.dirname(__file__), './settings.yaml')
try:
    config = load_config(config_path)
    PORT = config['local_port_python_services']
except (FileNotFoundError, RuntimeError, KeyError) as e:
    print(f"Configuration error: {e}")
    exit(1)  # Exit the program with an error code

class SimpleHTTPRequestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        parsed_url = urlparse(self.path)
        query_params = parse_qs(parsed_url.query)
        path = parsed_url.path
        if path == '/meta':
            id = query_params['id'][0]
            data = ytmusic.get_song(id)
            vtype = data['videoDetails']['musicVideoType'].lower()
            if "atv" in vtype:
                vtype = "atv"
            else:
                vtype = "omv"
            response = {
                'title': data['videoDetails']['title'],
                'author': data['videoDetails']['author'],
                'image': data['videoDetails']['thumbnail']['thumbnails'][-1]['url'],
                'type': vtype
            }
            self.send_response(200)
            self.end_headers()
            self.wfile.write(json.dumps(response).encode('utf-8'))
        elif path == '/playlist':
            id = query_params['id'][0]
            tracks = ytmusic.get_playlist(playlistId=id, limit=None)
            vid = []
            for t in tracks["tracks"]:
                vid.append({'id':t["videoId"]})
            response = {'tracks': vid}
            self.send_response(200)
            self.end_headers()
            self.wfile.write(json.dumps(response).encode('utf-8'))
        elif path == '/kill':
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b'Server is shutting down...')
            shutdown_flag.set()
        else:
            self.send_response(404)
            self.end_headers()
            self.wfile.write(b'Not Found')

def run_server():
    with socketserver.TCPServer(("0.0.0.0", PORT), SimpleHTTPRequestHandler) as httpd:
        print(f"Serving on port {PORT}")
        while not shutdown_flag.is_set():
            httpd.handle_request()

# Run the server in a separate thread
server_thread = threading.Thread(target=run_server)
server_thread.start()

# Wait for the shutdown signal
try:
    while not shutdown_flag.is_set():
        time.sleep(1)
finally:
    print("Server has been shut down.")
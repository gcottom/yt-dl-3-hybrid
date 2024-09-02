from ytmusicapi import YTMusic
import json

ytmusic = YTMusic()

def lambda_handler(event, context):
    tracks = ytmusic.get_playlist(id=event['queryStringParameters']['id'], limit=None)
    vid = []
    for t in tracks["tracks"]:
        vid.append({'id':t["videoId"]})
    return {
        "statusCode": 200,
        "body": json.dumps({
            'tracks': vid
        })
    }
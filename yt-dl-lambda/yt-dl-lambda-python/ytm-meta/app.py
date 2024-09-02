from ytmusicapi import YTMusic
import json

ytmusic = YTMusic()

def lambda_handler(event, context):
    id = event['queryStringParameters']['id']
    data = ytmusic.get_song(id)
    vtype = data['videoDetails']['musicVideoType'].lower()
    if "atv" in vtype:
        vtype = "atv"
    else:
        vtype = "omv"
    return {
        "statusCode": 200,
        "body": json.dumps({
            'title': data['videoDetails']['title'],
            'author': data['videoDetails']['author'],
            'image': data['videoDetails']['thumbnail']['thumbnails'][-1]['url'],
            'type': vtype
        })
    }
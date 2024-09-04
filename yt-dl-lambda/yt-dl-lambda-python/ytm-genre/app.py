import os
os.environ['LIBROSA_CACHE_DIR'] = '/tmp/librosa_cache'
os.environ['NUMBA_CACHE_DIR'] = '/tmp/numba_cache'
from musicnn.tagger import top_tags
import boto3
import botocore
import json

s3 = boto3.resource('s3')
sqs_client = boto3.client("sqs")
track_table = boto3.resource('dynamodb').Table('YTDL3Tracks')

def lambda_handler(event, context):
    batch_item_failures = []
    sqs_batch_response = {}
    for record in event['Records']:
        try:
            id = str(record['body'])
            conv = id + ".mp3"
            track = "/tmp/" + conv
            s3.meta.client.download_file(os.environ.get("AWS_DOWNLOADS_BUCKET"), conv, track)
            tops = 5
            
            t = top_tags(track, model='MSD_musicnn', topN=tops, print_tags=False)
            v = top_tags(track, model='MSD_vgg', topN=tops, print_tags=False)
            x = top_tags(track, model='MTT_musicnn', topN=tops, print_tags=False)
            y = top_tags(track, model='MTT_vgg', topN=tops, print_tags=False)
            print(t)
            print(v)
            print(x)
            print(y)
            z = {}
            w = tops
            for l in t:
                z[l] = w
                w=w-1
            w=tops
            for l in v:
                if l in z: 
                    z[l] = z[l] +w
                else:
                    z[l] = w
                w=w-1
            w=tops
            for l in x:
                if l in z: 
                    z[l] = z[l] +w
                else:
                    z[l] = w
                w=w-1
            w=tops
            for l in y:
                if l in z: 
                    z[l] = z[l] +w
                else:
                    z[l] = w
                w=w-1
            s = sorted(z.items(), key=lambda x:x[1], reverse=True)
            z = dict(s)
            print(z)
            g = ['classical', 'techno', 'strings', 'drums', 'electronic', 'rock', 'piano', 'ambient', 'violin', 'vocal', 'synth', 'indian', 'opera', 'harpsichord', 'flute', 'pop', 'sitar', 'classic', 'choir', 'new age', 'dance', 'harp', 'cello', 'country', 'metal', 'choral', 'alternative', 'indie', '00s', 'alternative rock', 'jazz', 'chillout', 'classic rock', 'soul', 'indie rock', 'Mellow', 'electronica', '80s', 'folk', '90s', 'chill', 'instrumental', 'punk', 'oldies', 'blues', 'hard rock', 'acoustic', 'experimental', 'Hip-Hop', '70s', 'party', 'easy listening', 'funk', 'electro', 'heavy metal', 'Progressive rock', '60s', 'rnb', 'indie pop', 'sad', 'House']
            for k in z:
                if k in g:
                    z = k
                    break
            if z == dict(s):
                z = list(z.keys())[0]
            print(z.title())
            try:
                sqs_client.send_message(
                    QueueUrl="https://sqs." + os.environ.get('AWS_REGION') + ".amazonaws.com/" + os.environ.get('AWS_ACCOUNT_ID') + "/yt-dl-3-meta",
                    MessageBody=json.dumps({
                        "id": id,
                        "genre": z.title()
                    })
                )
            except boto3.ClientError as err:
                raise
        except Exception as e:
            batch_item_failures.append({"itemIdentifier": record['messageId']})  
    sqs_batch_response["batchItemFailures"] = batch_item_failures
    return sqs_batch_response
# Databox-Instagram-Driver
## Data Format
Data for the Instagram driver is gathered using [an Instagram Scraper](https://github.com/rarcega/instagram-scraper).
A scraper is being used instead of the Instagram API because it is being scheduled for deprecation.

All data is updated currently every 30 seconds (testing purposes).

## Data Stores
The Instagram driver has two datastores

### Credential Datastore
The credential datastore is a key-value store **(KVStore)** which holds the users username and password. This allows the user to loging with saved credentials if they wish. The content type that is stored inside the store is text **(ContentTypeText)**.

The credential store ID is: ***InstagramCred***

### Image Datastore

The image datastore is a key-value store **(KVStore)** which contains the metadata json of all the stored images inside the store, along with individual JSON objects of each image. The content that is stored inside the store is in JSON format **(ContentTypeJSON)**

The metadata that is stored can be accessed with the key: ***metadata***

The image datastore ID is: ***InstagramDatastore***

### Image Data example
```
{"StoreID":"44711812_294526347937561_4787646456149181055_n.jpg",
"dimensions":{"width":1080,"height":1080},
"display_url":"https://scontent-lhr3-1.cdninstagram.com/vp/3db837bacefb3e16131dfd79bf72276f/5C8B709F/t51.2885-15/e35/44711812_294526347937561_4787646456149181055_n.jpg?se=7\u0026ig_cache_key=MTkxMzIyMzY4NjMyOTA5MDMyMA%3D%3D.2",
"id":"1913223686329090320",
"is_video":false,
"tags":[],
"taken_at_timestamp":1542294058,
"edge_media_preview_like":{"count":0},
"edge_media_to_caption":{"edges":[{"node":{"text":"Easterlings 2"}}]},
"edge_media_to_comment":{"count":0}}
```

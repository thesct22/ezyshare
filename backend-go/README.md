# Server for signalling the details of both devices

[![Heroku](https://heroku-badge.herokuapp.com/?app=heroku-badge&style=flat)](https://guarded-atoll-76755.herokuapp.com/)

Endpoints are:
* /recv
* /send
* /make & /

## Uses Websockets to receive device details to share to each other.

Details include:

1. Unique ID for connections
2. Details of the connected devices and connection status
3. Details of ICE Candidates so devices can connect with each other
4. Whether file is uploaded and ready to be shared
5. Whether receiver(s) is/are connected and ready to receiveg
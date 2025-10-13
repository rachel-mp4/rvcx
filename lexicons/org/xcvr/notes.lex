org.xcvr
  actor
    profile: record
      displayName?: string, bytes<=640, chars<=64
      defaultNick?: string, bytes<=16
      status?: string, bytes<=6400, chars<=640
      avatar?: blob
      color?: int, [0 16777215]

    profileView: def
      did: did
      handle: handle
      displayName?: string, bytes<=640, chars<=64
      status?: string, bytes<=6400, chars<=640
      avatar?: blob
      color?: int, [0 16777215]
      defaultNick?: string, bytes<=16

    resolveChannel: query
      params
        union
          handle: handle
          rkey: string
        |
          did: did
          rkey: string
        |
          uri: uri
      output
        url: string
        uri?: uri

    getProfile: query
      params
        union
          handle: handle
        |
          did: did
      output
        profile: profileView

    getLastSeen: query
      params
        union
          handle: handle
        |
          did: did
      output
        where?: uri
        when?: date

  feed
    channel: record
      title: string, bytes<=640, chars<=64 
      topic?: string, bytes<=2560, chars<=256
      createdAt: string
      host: handle

    sub: record
      channelUri: uri

    channelView?: def
      uri: uri
      host: handle
      creator: org.xcvr.actor.profileView
      title: string, bytes<=640, chars<=64
      topic?: string, bytes<=2560, chars<=256
      connectedCount?: int [0
      createdAt?: date

    getChannels?: query
      params 
        limit?: int, [0 100], default=50
        cursor?: string
      output
        channels: array
          channelView
        cursor?: string

  lrc
    message: record
      signetURI: uri
      body: string
      nick?: string, bytes<=16
      color?: int, [0 16777215]
      postedAt?: date

    signet: record
      channelURI: uri
      lrcID: int, [0 4294967295]
      author: string
      startedAt?: date

    media: record
      signetURI: uri
      union
        image
      |
        video
      nick?: string, bytes<=16
      color?: int, [0 16777215]
      postedAt?: date

    image: def
      alt: string
      blob: blob
      aspectRatio?: aspectRatio

    aspectRatio: def
      width: int
      height: int

    video: def
      alt: string
      blob: blob
      # captions?
    
    messageView: def
      uri: uri
      author: org.xcvr.lrc.profileView
      body: string
      nick?: string, bytes<=16
      color?: int, [0 16777215]
      signetURI: uri
      postedAt: date

    signetView: def
      uri: uri
      issuer: handle
      channelURI: uri
      lrcID: int, [0 4294967295]
      authorHandle: string
      startedAt?: date

    mediaView: def
      uri: uri
      author: org.xcvr.lrc.profileView
      union
        image
      |
        video
      nick?: string, bytes<=16
      color?: int, [0 16777215]
      signetURI: uri
      postedAt: date

    signedMessageView: def
      uri: uri
      author: org.xcvr.lrc.profileView
      body: string
      nick?: string, bytes<=16
      color?: int, [0 16777215]
      signet: signetView
      postedAt: date

    signedMediaView: def
      uri: uri
      author: org.xcvr.lrc.profileView
      union
        image
      |
        video
      nick?: string, bytes<=16
      color?: int, [0 16777215]
      signet: signetView
      postedAt: date

    getMessages: query
      params 
        limit?: int, [0 100], default=50
        cursor?: string
      output
        messages: array
          union
            signedMessageView
          |
            signedMediaView
        cursor?: string
    
    subscribeLexStream: subscription
      params
        uri: uri
      message
        union
          messageView
        |
          signetView 
        |
          mediaView


        




      



      
        


      



        
      

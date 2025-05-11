# Communication - Pub Sub

The pubsub communications are provided by NATS and the library located under src/common/signal/

### Client Configuration:

Client configuration operates in one of 2 modes.
- Server Mode: For clients operating the consensus engine, doing rpc monitoring and providng powerdns api endpoints.
- Client Mode: For clients wanting to query information from the monitoring endpoints related to current or past operations.

### Server Mode

In server mode, the client will subscribe to and listen for server mode related subjects.

### Client Mode

In client mode, the client will be able to send and listen for client related subjects.

## Consensus Communication
#### Server Mode
###### Subscriptions
- subject: consensus.propose 
- subject: consensus.vote
- subject: consensus.finalize
- subject: consensus.cluster
- subject: consensus.members
###### Message Subjects
- subject: consensus.propose
- subject: consensus.vote
- subject: consensus.finalize
- subject: consensus.cluster
- subject: consensus.memberList

#### Client Mode
###### Subscriptions
- subject: consensus.memberList

###### Message Subjects
- subject: consensus.getMembers

## Billing Communication
#### Server Mode
###### Subscriptions
- subject: billing.getData
###### Message Subjects
- subject: billing.data

#### Client Mode
###### Subscriptions
- subject: billing.data
###### Message Subjects
- subject: billing.getData

## Stats Communication
#### Server Mode
###### Subscriptions
- subject: stats.getData
###### Message Subjects
- subject: stats.data

#### Client Mode
###### Subscriptions
- subject: stats.data
###### Message Subjects
- subject: stats.getData

## Usage Communication
#### Server Mode
###### Subscriptions
- subject: usage.getData
###### Message Subjects
- subject: usage.data

#### Client Mode
###### Subscriptions
- subject: usage.data
###### Message Subjects
- subject: usage.getData

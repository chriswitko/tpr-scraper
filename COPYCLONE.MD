Copy vs Clone
You may have noticed that when using the mgo Mongodb library. We create a database session, which is passed into our handlers, but on each request we call a method which clones that session and passes it into our repository code.

Effectively, aside from spawning the first connection to the database, we never touch the 'master session', we called session.Clone() each time we want to make a call to the datastore. As I mentioned briefly in the code comments, but I think it's worth re-iterating this in some detail, if you use the master session, you are re-using the same socket. Which means your queries may become blocked by other queries and have to wait for operations to finish on this socket. Which is pointless in a language which supports concurrency.

So to avoid blocking requests, mgo allows you to Copy() or Clone() a session, so that you have a concurrent connection for each request. You will notice that I've mentioned Copy and Clone methods, these are very similar, but have a subtle, but important difference. Clone re-uses the same socket as master. Which reduces the overhead of spawning an entirely new socket. Which is perfect for fast write performance. However, longer operations, such as more complex queries or big data jobs etc, may cause blocking in other go routines attempting to use this socket.

Generally speaking, you're probably better off with Clone for purposes such as ours.

- https://ewanvalentine.io/microservices-in-golang-part-3/
# Observable Streams on Golang
Package provides an API for asynchronous programming with observable streams like [Durt's streams](https://api.dartlang.org/stable/2.1.0/dart-async/Stream-class.html) or like ReacitveX pattern. 

Stream is a source of asynchronous data events.

ReactiveX is a new, alternative way of asynchronous programming to callbacks, promises and deferred. It is about processing streams of events or items, with events being any occurrences or changes within the system.

More about Reactive pattern you can read on [reactivex.io](http://reactivex.io/).

## Lisense 
MIT License

## Why don't use RxGo?
The `https://github.com/ReactiveX/RxGo` package doesn't provide broadcast Streams, which is useful for certain purposes.

There are two kinds of streams: **"Single-subscription"** streams and **"broadcast"** streams.

A single-subscription stream allows only a single listener during the whole lifetime of the stream. It doesn't start generating events until it has a listener, and it stops sending events when the listener is unsubscribed, even if the source of events could still provide more.

Listening twice on a single-subscription stream is not allowed, even after the first subscription has been canceled.

A **broadcast stream** allows any number of listeners, and it fires its events when they are ready, whether there are listeners or not.

## How to use
Broadcast streams are used for independent events/observers. To create broadcasting streams just use:
```golang
    myStream := streams.NewStream() 
```
To put events use `Add` and `AddError` methods:
```golang
    myStream.Add(someEvent)
    // or 
    myStream.AddError(someError)
```
Listeners must subscribe to a stream:
```golang
    eventHandler := func(value interface{}) {
        // some code on event
    }

    myStream.Listen(handler)
    // myStream.Listen(anotherHandler)
```
Streams can be transformed:
```golang
    // filter example with first item:
    filter := func(x interface{}) bool {
        return x.(int) > 3
    }
    myStream.Filter(filter).First().Listen(myHandler)

    // or map exmaple
    myMap := func(x interface{}) interface{} {
        return fmt.Sprintf("Hello, %d number", x.(int))
    }
    myStream.First().Map(myMap).Listen(myHandler2)
```

### Attention
Be aware that all stream transformed methods return new stream. You should subscribe your event listeners respectively.
 ```golang
     // DON'T:
     myStream.Listen(myHandler).Filter(filter).Take(5)
 
     // DO:
     myStream.Filter(filter).Take(5).Listen(myHandler)
 ```

## Contributing
You are welcome! Github issues is the best place for that's purposes.
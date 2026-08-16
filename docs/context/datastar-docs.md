Datastar Docs 🚀

How this app uses Datastar (signals, patches, what not to @get): `docs/context/datastar-app.md`

> HTML version available at https://data-star.dev/docs

# Getting Started

Datastar simplifies frontend development, allowing you to build backend-driven, interactive UIs using a [hypermedia-first](https://hypermedia.systems/hypermedia-a-reintroduction/) approach that extends and enhances HTML.

Datastar provides backend reactivity like [htmx](https://htmx.org/) and frontend reactivity like [Alpine.js](https://alpinejs.dev/) in a lightweight frontend framework that doesn’t require any npm packages or other dependencies. It provides two primary functions:
. Modify the DOM and state by sending events from your backend.
. Build reactivity into your frontend using standard `data-*` HTML attributes.

## Installation 

The quickest way to use Datastar is to include it using a `script` tag that fetches it from a CDN.

```
<script type="module" src="https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.1/bundles/datastar.js"></script>
```

Hosting the file yourself is recommended. Download the [script](https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.1/bundles/datastar.js) or create your own bundle using the [bundler](https://data-star.dev/pro/bundler), then include it from the appropriate path.

```
<script type="module" src="/path/to/datastar.js"></script>
```

To import Datastar using a package manager such as npm, Deno, or Bun, you can use an import statement.

```
// @ts-expect-error (only required for TypeScript projects)
import 'https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.1/bundles/datastar.js'
```

## `data-*` 

At the core of Datastar are `data-*` HTML attributes (hence the name). They allow you to add reactivity to your frontend and interact with your backend in a declarative way.

> The Datastar [VSCode extension](https://marketplace.visualstudio.com/items?itemName=starfederation.datastar-vscode) and [IntelliJ plugin](https://plugins.jetbrains.com/plugin/26072-datastar-support) provide autocompletion for all available `data-*` attributes.

The [`data-on`](https://data-star.dev/reference/attributes#data-on) attribute can be used to attach an event listener to an element and execute an expression whenever the event is triggered. The value of the attribute is a [Datastar expression](https://data-star.dev/guide/datastar_expressions) in which JavaScript can be used.

```
<button data-on:click="alert('I’m sorry, Dave. I’m afraid I can’t do that.')">
    Open the pod bay doors, HAL.
</button>
```

Demo

Open the pod bay doors, HAL.

We’ll explore more data attributes in the [next section of the guide](https://data-star.dev/guide/reactive_signals).

## Patching Elements 

With Datastar, the backend *drives* the frontend by **patching** (adding, updating and removing) HTML elements in the DOM.

Datastar receives elements from the backend and manipulates the DOM using a morphing strategy (by default). Morphing ensures that only modified parts of the DOM are updated, and that only data attributes that have changed are [reapplied](https://data-star.dev/reference/attributes#attribute-evaluation-order), preserving state and improving performance.

Datastar provides [actions](https://data-star.dev/reference/actions#backend-actions) for sending requests to the backend. The [`@get()`](https://data-star.dev/reference/actions#get) action sends a `GET` request to the provided URL using a [fetch](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API) request.

```
<button data-on:click="@get('/endpoint')">
    Open the pod bay doors, HAL.
</button>
<div id="hal"></div>
```

> Actions in Datastar are helper functions that have the syntax `@actionName()`. Read more about actions in the [reference](https://data-star.dev/reference/actions).

If the response has a `content-type` of `text/html`, the top-level HTML elements will be morphed into the existing DOM based on the element IDs.

```
<div id="hal">
    I’m sorry, Dave. I’m afraid I can’t do that.
</div>
```

We call this a “Patch Elements” event because multiple elements can be patched into the DOM at once.

Demo

Open the pod bay doors, HAL. `Waiting for an order...`

In the example above, the DOM must contain an element with a `hal` ID in order for morphing to work. Other [patching strategies](https://data-star.dev/reference/sse_events#datastar-patch-elements) are available, but morph is the best and simplest choice in most scenarios.

If the response has a `content-type` of `text/event-stream`, it can contain zero or more [SSE events](https://data-star.dev/reference/sse_events). The example above can be replicated using a `datastar-patch-elements` SSE event. Note that SSE events must be followed by two newline characters.

```
event: datastar-patch-elements
data: elements <div id="hal">
data: elements     I’m sorry, Dave. I’m afraid I can’t do that.
data: elements </div>

```

Because we can send as many events as we want in a stream, and because it can be a long-lived connection, we can extend the example above to first send HAL’s response and then, after a few seconds, reset the text.

```
event: datastar-patch-elements
data: elements <div id="hal">
data: elements     I’m sorry, Dave. I’m afraid I can’t do that.
data: elements </div>

event: datastar-patch-elements
data: elements <div id="hal">
data: elements     Waiting for an order...
data: elements </div>

```

Demo

Open the pod bay doors, HAL. `Waiting for an order...`

Here’s the code to generate the SSE events above using the SDKs.

```
;; Import the SDK's api and your adapter
(require
 '[starfederation.datastar.clojure.api :as d*]
 '[starfederation.datastar.clojure.adapter.http-kit :refer [->sse-response on-open]])

;; in a ring handler
(defn handler [request]
  ;; Create an SSE response
  (->sse-response request
                  {on-open
                   (fn [sse]
                     ;; Patches elements into the DOM
                     (d*/patch-elements! sse
                                         "<div id=\"hal\">I’m sorry, Dave. I’m afraid I can’t do that.</div>")
                     (Thread/sleep 1000)
                     (d*/patch-elements! sse
                                         "<div id=\"hal\">Waiting for an order...</div>"))}))
```

```
using StarFederation.Datastar.DependencyInjection;

// Adds Datastar as a service
builder.Services.AddDatastar();

app.MapGet("/", async (IDatastarService datastarService) =>
{
    // Patches elements into the DOM.
    await datastarService.PatchElementsAsync(@"<div id=""hal"">I’m sorry, Dave. I’m afraid I can’t do that.</div>");

    await Task.Delay(TimeSpan.FromSeconds(1));

    await datastarService.PatchElementsAsync(@"<div id=""hal"">Waiting for an order...</div>");
});
```

```
import (
    "github.com/starfederation/datastar-go/datastar"
    time
)

// Creates a new `ServerSentEventGenerator` instance.
sse := datastar.NewSSE(w,r)

// Patches elements into the DOM.
sse.PatchElements(
    `<div id="hal">I’m sorry, Dave. I’m afraid I can’t do that.</div>`
)

time.Sleep(1 * time.Second)

sse.PatchElements(
    `<div id="hal">Waiting for an order...</div>`
)
```

```
import starfederation.datastar.utils.ServerSentEventGenerator;

// Creates a new `ServerSentEventGenerator` instance.
AbstractResponseAdapter responseAdapter = new HttpServletResponseAdapter(response);
ServerSentEventGenerator generator = new ServerSentEventGenerator(responseAdapter);

// Patches elements into the DOM.
generator.send(PatchElements.builder()
    .data("<div id=\"hal\">I’m sorry, Dave. I’m afraid I can’t do that.</div>")
    .build()
);

Thread.sleep(1000);

generator.send(PatchElements.builder()
    .data("<div id=\"hal\">Waiting for an order...</div>")
    .build()
);
```

```
val generator = ServerSentEventGenerator(response)

generator.patchElements(
    elements = """<div id="hal">I’m sorry, Dave. I’m afraid I can’t do that.</div>""",
)

Thread.sleep(ONE_SECOND)

generator.patchElements(
    elements = """<div id="hal">Waiting for an order...</div>""",
)
```

```
use starfederation\datastar\ServerSentEventGenerator;

// Creates a new `ServerSentEventGenerator` instance.
$sse = new ServerSentEventGenerator();

// Patches elements into the DOM.
$sse->patchElements(
    '<div id="hal">I’m sorry, Dave. I’m afraid I can’t do that.</div>'
);

sleep(1);

$sse->patchElements(
    '<div id="hal">Waiting for an order...</div>'
);
```

```
from datastar_py import ServerSentEventGenerator as SSE
from datastar_py.sanic import datastar_response

@app.get('/open-the-bay-doors')
@datastar_response
async def open_doors(request):
    yield SSE.patch_elements('<div id="hal">I’m sorry, Dave. I’m afraid I can’t do that.</div>')
    await asyncio.sleep(1)
    yield SSE.patch_elements('<div id="hal">Waiting for an order...</div>')
```

```
require 'datastar'

# Create a Datastar::Dispatcher instance

datastar = Datastar.new(request:, response:)

# In a Rack handler, you can instantiate from the Rack env
# datastar = Datastar.from_rack_env(env)

# Start a streaming response
datastar.stream do |sse|
  # Patches elements into the DOM.
  sse.patch_elements %(<div id="hal">I’m sorry, Dave. I’m afraid I can’t do that.</div>)

  sleep 1
  
  sse.patch_elements %(<div id="hal">Waiting for an order...</div>)
end
```

```
use async_stream::stream;
use datastar::prelude::*;
use std::thread;
use std::time::Duration;

Sse(stream! {
    // Patches elements into the DOM.
    yield PatchElements::new("<div id='hal'>I’m sorry, Dave. I’m afraid I can’t do that.</div>").into();

    thread::sleep(Duration::from_secs(1));
    
    yield PatchElements::new("<div id='hal'>Waiting for an order...</div>").into();
})
```

```
// Creates a new `ServerSentEventGenerator` instance (this also sends required headers)
ServerSentEventGenerator.stream(req, res, (stream) => {
    // Patches elements into the DOM.
    stream.patchElements(`<div id="hal">I’m sorry, Dave. I’m afraid I can’t do that.</div>`);

    setTimeout(() => {
        stream.patchElements(`<div id="hal">Waiting for an order...</div>`);
    }, 1000);
});
```

> In addition to your browser’s dev tools, the [Datastar Inspector](https://data-star.dev/pro#datastar-inspector) can be used to monitor and inspect SSE events received by Datastar.

We’ll cover event streams and [SSE events](https://data-star.dev/reference/sse_events) in more detail [later in the guide](https://data-star.dev/guide/backend_requests), but as you can see, they are just plain text events with a special syntax, made simpler by the [SDKs](https://data-star.dev/reference/sdks).

# Reactive Signals

In a hypermedia approach, the backend drives state to the frontend and acts as the primary source of truth. It’s up to the backend to determine what actions the user can take next by patching appropriate elements in the DOM.

Sometimes, however, you may need access to frontend state that’s driven by user interactions. Click, input and keydown events are some of the more common user events that you’ll want your frontend to be able to react to.

Datastar uses *signals* to manage frontend state. You can think of signals as reactive variables that automatically track and propagate changes in and to [Datastar expressions](https://data-star.dev/guide/datastar_expressions). Signals are denoted using the `$` prefix.

## Data Attributes 

Datastar allows you to add reactivity to your frontend and interact with your backend in a declarative way using [custom `data-*` attributes](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Global_attributes/data-*).

> The Datastar [VSCode extension](https://marketplace.visualstudio.com/items?itemName=starfederation.datastar-vscode) and [IntelliJ plugin](https://plugins.jetbrains.com/plugin/26072-datastar-support) provide autocompletion for all available `data-*` attributes.

### `data-bind` 

The [`data-bind`](https://data-star.dev/reference/attributes#data-bind) attribute sets up two-way data binding on any HTML element that receives user input or selections. These include `input`, `textarea`, `select`, `checkbox` and `radio` elements, as well as web components whose value can be made reactive.

```
<input data-bind:foo />
```

This creates a new signal that can be called using `$foo`, and binds it to the element’s value. If either is changed, the other automatically updates.

You can accomplish the same thing passing the signal name as a *value*. This syntax can be more convenient to use with some templating languages.

```
<input data-bind="foo" />
```

According to the [HTML spec](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes/data-*), all [`data-*`](https://developer.mozilla.org/en-US/docs/Web/HTML/How_to/Use_data_attributes) attributes are case-insensitive. When Datastar processes these attributes, hyphenated names are automatically converted to camel case by removing hyphens and uppercasing the letter following each hyphen. For example, `data-bind:foo-bar` creates a signal named `$fooBar`.

```
<!-- Both of these create the signal `$fooBar` -->
<input data-bind:foo-bar />
<input data-bind="fooBar" />
```

Read more about [attribute casing](https://data-star.dev/reference/attributes#attribute-casing) in the reference.

### `data-text` 

The [`data-text`](https://data-star.dev/reference/attributes#data-text) attribute sets the text content of an element to the value of a signal. The `$` prefix is required to denote a signal.

```
<input data-bind:foo-bar />
<div data-text="$fooBar"></div>
```

Demo

```

```

The value of the `data-text` attribute is a [Datastar expression](https://data-star.dev/guide/datastar_expressions) that is evaluated, meaning that we can use JavaScript in it.

```
<input data-bind:foo-bar />
<div data-text="$fooBar.toUpperCase()"></div>
```

Demo

```

```

### `data-computed` 

The [`data-computed`](https://data-star.dev/reference/attributes#data-computed) attribute creates a new signal that is derived from a reactive expression. The computed signal is read-only, and its value is automatically updated when any signals in the expression are updated.

```
<input data-bind:foo-bar />
<div data-computed:repeated="$fooBar.repeat(2)" data-text="$repeated"></div>
```

This results in the `$repeated` signal’s value always being equal to the value of the `$fooBar` signal repeated twice. Computed signals are useful for memoizing expressions containing other signals.

Demo

```

```

### `data-show` 

The [`data-show`](https://data-star.dev/reference/attributes#data-show) attribute can be used to show or hide an element based on whether an expression evaluates to `true` or `false`.

```
<input data-bind:foo-bar />
<button data-show="$fooBar != ''">
    Save
</button>
```

This results in the button being visible only when the input value is *not* an empty string. This could also be shortened to `data-show="$fooBar"`.

Demo

Save

Since the button is visible until Datastar processes the `data-show` attribute, it’s a good idea to set its initial style to `display: none` to prevent a flash of unwanted content.

```
<input data-bind:foo-bar />
<button data-show="$fooBar != ''" style="display: none">
    Save
</button>
```

### `data-class` 

The [`data-class`](https://data-star.dev/reference/attributes#data-class) attribute allows us to add or remove an element’s class based on an expression.

```
<input data-bind:foo-bar />
<button data-class:success="$fooBar != ''">
    Save
</button>
```

If the expression evaluates to `true`, the `success` class is added to the element, otherwise it is removed.

Demo

Save

Unlike the `data-bind` attribute, in which hyphenated names are converted to camel case, the `data-class` attribute converts the class name to kebab case. For example, `data-class:font-bold` adds or removes the `font-bold` class.

```
<button data-class:font-bold="$fooBar == 'strong'">
    Save
</button>
```

The `data-class` attribute can also be used to add or remove multiple classes from an element using a set of key-value pairs, where the keys represent class names and the values represent expressions.

```
<button data-class="{success: $fooBar != '', 'font-bold': $fooBar == 'strong'}">
    Save
</button>
```

Note how the `font-bold` key must be wrapped in quotes because it contains a hyphen.

### `data-attr` 

The [`data-attr`](https://data-star.dev/reference/attributes#data-attr) attribute can be used to bind the value of any HTML attribute to an expression.

```
<input data-bind:foo />
<button data-attr:disabled="$foo == ''">
    Save
</button>
```

This results in a `disabled` attribute being given the value `true` whenever the input is an empty string.

Demo

Save

The `data-attr` attribute also converts the attribute name to kebab case, since HTML attributes are typically written in kebab case. For example, `data-attr:aria-hidden` sets the value of the `aria-hidden` attribute.

```
<button data-attr:aria-hidden="$foo">Save</button>
```

The `data-attr` attribute can also be used to set the values of multiple attributes on an element using a set of key-value pairs, where the keys represent attribute names and the values represent expressions.

```
<button data-attr="{disabled: $foo == '', 'aria-hidden': $foo}">Save</button>
```

Note how the `aria-hidden` key must be wrapped in quotes because it contains a hyphen.

### `data-signals` 

Signals are globally accessible from anywhere in the DOM. So far, we’ve created signals on the fly using `data-bind` and `data-computed`. If a signal is used without having been created, it will be created automatically and its value set to an empty string.

Another way to create signals is using the [`data-signals`](https://data-star.dev/reference/attributes#data-signals) attribute, which patches (adds, updates or removes) one or more signals into the existing signals.

```
<div data-signals:foo-bar="1"></div>
```

Signals can be nested using dot-notation.

```
<div data-signals:form.baz="2"></div>
```

Like the `data-bind` attribute, hyphenated names used with `data-signals` are automatically converted to camel case by removing hyphens and uppercasing the letter following each hyphen.

```
<div data-signals:foo-bar="1"
     data-text="$fooBar"
></div>
```

The `data-signals` attribute can also be used to patch multiple signals using a set of key-value pairs, where the keys represent signal names and the values represent expressions. Nested signals can be created using nested objects.

```
<div data-signals="{fooBar: 1, form: {baz: 2}}"></div>
```

### `data-on` 

The [`data-on`](https://data-star.dev/reference/attributes#data-on) attribute can be used to attach an event listener to an element and run an expression whenever the event is triggered.

```
<input data-bind:foo />
<button data-on:click="$foo = ''">
    Reset
</button>
```

This results in the `$foo` signal’s value being set to an empty string whenever the button element is clicked. This can be used with any valid event name such as `data-on:keydown`, `data-on:mouseover`, etc.

Demo

Reset

Custom events can also be used. Like the `data-class` attribute, the `data-on` attribute converts the event name to kebab case. For example, `data-on:custom-event` listens for the `custom-event` event.

```
<div data-on:my-event="$foo = ''">
    <input data-bind:foo />
</div>
```

These are just *some* of the attributes available in Datastar. For a complete list, see the [attribute reference](https://data-star.dev/reference/attributes).

## Frontend Reactivity 

Datastar’s data attributes enable declarative signals and expressions, providing a simple yet powerful way to add reactivity to the frontend.

Datastar expressions are strings that are evaluated by Datastar [attributes](https://data-star.dev/reference/attributes) and [actions](https://data-star.dev/reference/actions). While they are similar to JavaScript, there are some important differences that are explained in the [next section of the guide](https://data-star.dev/guide/datastar_expressions).

```
<div data-signals:hal="'...'">
    <button data-on:click="$hal = 'Affirmative, Dave. I read you.'">
        HAL, do you read me?
    </button>
    <div data-text="$hal"></div>
</div>
```

Demo

HAL, do you read me?

```

```

See if you can figure out what the code below does based on what you’ve learned so far, *before* trying the demo below it.

```
<div
    data-signals="{response: '', answer: 'bread'}"
    data-computed:correct="$response.toLowerCase() == $answer"
>
    <div id="question">What do you put in a toaster?</div>
    <button data-on:click="$response = prompt('Answer:') ?? ''">BUZZ</button>
    <div data-show="$response != ''">
        You answered “<span data-text="$response"></span>”.
        <span data-show="$correct">That is correct ✅</span>
        <span data-show="!$correct">
        The correct answer is “
        <span data-text="$answer"></span>
        ” 🤷
        </span>
    </div>
</div>
```

Demo

What do you put in a toaster?

BUZZ

You answered “”. That is correct ✅ The correct answer is “bread” 🤷

> The [Datastar Inspector](https://data-star.dev/pro#datastar-inspector) can be used to inspect and filter current signals and view signal patch events in real-time.

## Patching Signals 

Remember that in a hypermedia approach, the backend drives state to the frontend. Just like with elements, frontend signals can be **patched** (added, updated and removed) from the backend using [backend actions](https://data-star.dev/reference/actions#backend-actions).

```
<div data-signals:hal="'...'">
    <button data-on:click="@get('/endpoint')">
        HAL, do you read me?
    </button>
    <div data-text="$hal"></div>
</div>
```

If a response has a `content-type` of `application/json`, the signal values are patched into the frontend signals.

We call this a “Patch Signals” event because multiple signals can be patched (using [JSON Merge Patch RFC 7396](https://datatracker.ietf.org/doc/rfc7396/)) into the existing signals.

```
{"hal": "Affirmative, Dave. I read you."}
```

Demo

HAL, do you read me?

Reset

If the response has a `content-type` of `text/event-stream`, it can contain zero or more [SSE events](https://data-star.dev/reference/sse_events). The example above can be replicated using a `datastar-patch-signals` SSE event.

```
event: datastar-patch-signals
data: signals {hal: 'Affirmative, Dave. I read you.'}

```

Because we can send as many events as we want in a stream, and because it can be a long-lived connection, we can extend the example above to first set the `hal` signal to an “affirmative” response and then, after a second, reset the signal.

```
event: datastar-patch-signals
data: signals {hal: 'Affirmative, Dave. I read you.'}

// Wait 1 second

event: datastar-patch-signals
data: signals {hal: '...'}

```

Demo

HAL, do you read me?

Here’s the code to generate the SSE events above using the SDKs.

```
;; Import the SDK's api and your adapter
(require
  '[starfederation.datastar.clojure.api :as d*]
  '[starfederation.datastar.clojure.adapter.http-kit :refer [->sse-response on-open]])

;; in a ring handler
(defn handler [request]
  ;; Create an SSE response
  (->sse-response request
                  {on-open
                   (fn [sse]
                     ;; Patches signal.
                     (d*/patch-signals! sse "{hal: 'Affirmative, Dave. I read you.'}")
                     (Thread/sleep 1000)
                     (d*/patch-signals! sse "{hal: '...'}"))}))
```

```
using StarFederation.Datastar.DependencyInjection;

// Adds Datastar as a service
builder.Services.AddDatastar();

app.MapGet("/hal", async (IDatastarService datastarService) =>
{
    // Patches signals.
    await datastarService.PatchSignalsAsync(new { hal = "Affirmative, Dave. I read you" });

    await Task.Delay(TimeSpan.FromSeconds(3));

    await datastarService.PatchSignalsAsync(new { hal = "..." });
});
```

```
import (
    "github.com/starfederation/datastar-go/datastar"
)

// Creates a new `ServerSentEventGenerator` instance.
sse := datastar.NewSSE(w, r)

// Patches signals
sse.PatchSignals([]byte(`{hal: 'Affirmative, Dave. I read you.'}`))

time.Sleep(1 * time.Second)

sse.PatchSignals([]byte(`{hal: '...'}`))
```

```
import starfederation.datastar.utils.ServerSentEventGenerator;

// Creates a new `ServerSentEventGenerator` instance.
AbstractResponseAdapter responseAdapter = new HttpServletResponseAdapter(response);
ServerSentEventGenerator generator = new ServerSentEventGenerator(responseAdapter);

// Patches signals.
generator.send(PatchSignals.builder()
    .data("{\"hal\": \"Affirmative, Dave. I read you.\"}")
    .build()
);

Thread.sleep(1000);

generator.send(PatchSignals.builder()
    .data("{\"hal\": \"...\"}")
    .build()
);
```

```
val generator = ServerSentEventGenerator(response)

generator.patchSignals(
    signals = """{"hal": "Affirmative, Dave. I read you."}""",
)

Thread.sleep(ONE_SECOND)

generator.patchSignals(
    signals = """{"hal": "..."}""",
)
```

```
use starfederation\datastar\ServerSentEventGenerator;

// Creates a new `ServerSentEventGenerator` instance.
$sse = new ServerSentEventGenerator();

// Patches signals.
$sse->patchSignals(['hal' => 'Affirmative, Dave. I read you.']);

sleep(1);

$sse->patchSignals(['hal' => '...']);
```

```
from datastar_py import ServerSentEventGenerator as SSE
from datastar_py.sanic import datastar_response

@app.get('/do-you-read-me')
@datastar_response
async def open_doors(request):
    yield SSE.patch_signals({"hal": "Affirmative, Dave. I read you."})
    await asyncio.sleep(1)
    yield SSE.patch_signals({"hal": "..."})
```

```
require 'datastar'

# Create a Datastar::Dispatcher instance

datastar = Datastar.new(request:, response:)

# In a Rack handler, you can instantiate from the Rack env
# datastar = Datastar.from_rack_env(env)

# Start a streaming response
datastar.stream do |sse|
  # Patches signals
  sse.patch_signals(hal: 'Affirmative, Dave. I read you.')

  sleep 1
  
  sse.patch_signals(hal: '...')
end
```

```
use async_stream::stream;
use datastar::prelude::*;
use std::thread;
use std::time::Duration;

Sse(stream! {
    // Patches signals.
    yield PatchSignals::new("{hal: 'Affirmative, Dave. I read you.'}").into();

    thread::sleep(Duration::from_secs(1));
    
    yield PatchSignals::new("{hal: '...'}").into();
})
```

```
// Creates a new `ServerSentEventGenerator` instance (this also sends required headers)
ServerSentEventGenerator.stream(req, res, (stream) => {
    // Patches signals.
    stream.patchSignals({'hal': 'Affirmative, Dave. I read you.'});

    setTimeout(() => {
        stream.patchSignals({'hal': '...'});
    }, 1000);
});
```

> In addition to your browser’s dev tools, the [Datastar Inspector](https://data-star.dev/pro#datastar-inspector) can be used to monitor and inspect SSE events received by Datastar.

We’ll cover event streams and [SSE events](https://data-star.dev/reference/sse_events) in more detail [later in the guide](https://data-star.dev/guide/backend_requests), but as you can see, they are just plain text events with a special syntax, made simpler by the [SDKs](https://data-star.dev/reference/sdks).

# Datastar Expressions

Datastar expressions are strings that are evaluated by `data-*` attributes. While they are similar to JavaScript, there are some important differences that make them more powerful for declarative hypermedia applications.

## Datastar Expressions 

The following example outputs `1` because we’ve defined `foo` as a signal with the initial value `1`, and are using `$foo` in a `data-*` attribute.

```
<div data-signals:foo="1">
    <div data-text="$foo"></div>
</div>
```

A variable `el` is available in every Datastar expression, representing the element that the attribute is attached to.

```
<div data-text="el.offsetHeight"></div>
```

When Datastar evaluates the expression `$foo`, it first converts it to the signal value, and then evaluates that expression in a sandboxed context. This means that JavaScript can be used in Datastar expressions.

```
<div data-text="$foo.length"></div>
```

JavaScript operators are also available in Datastar expressions. This includes (but is not limited to) the ternary operator `?:`, the logical OR operator `||`, and the logical AND operator `&&`. These operators are helpful in keeping Datastar expressions terse.

```
// Output one of two values, depending on the truthiness of a signal
<div data-text="$landingGearRetracted ? 'Ready' : 'Waiting'"></div>

// Show a countdown if the signal is truthy or the time remaining is less than 10 seconds
<div data-show="$landingGearRetracted || $timeRemaining < 10">
    Countdown
</div>

// Only send a request if the signal is truthy
<button data-on:click="$landingGearRetracted && @post('/launch')">
    Launch
</button>
```

Multiple statements can be used in a single expression by separating them with a semicolon.

```
<div data-signals:foo="1">
    <button data-on:click="$landingGearRetracted = true; @post('/launch')">
        Force launch
    </button>
</div>
```

Expressions may span multiple lines, but a semicolon must be used to separate statements. Unlike JavaScript, line breaks alone are not sufficient to separate statements.

```
<div data-signals:foo="1">
    <button data-on:click="
        $landingGearRetracted = true; 
        @post('/launch')
    ">
        Force launch
    </button>
</div>
```

## Using JavaScript 

Most of your JavaScript logic should go in `data-*` attributes, since reactive signals and actions only work in [Datastar expressions](https://data-star.dev/guide/datastar_expressions).

> Caution: if you find yourself trying to do too much in Datastar expressions, **you are probably overcomplicating it™**.

Any JavaScript functionality you require that cannot belong in `data-*` attributes should be extracted out into [external scripts](#external-scripts) or, better yet, [web components](#web-components).

> Always encapsulate state and send **props down, events up**.

### External Scripts 

When using external scripts, you should pass data into functions via arguments and return a result. Alternatively, listen for custom events dispatched from them (props down, events up).

In this way, the function is encapsulated – all it knows is that it receives input via an argument, acts on it, and optionally returns a result or dispatches a custom event – and `data-*` attributes can be used to drive reactivity.

```
<div data-signals:result>
    <input data-bind:foo 
        data-on:input="$result = myfunction($foo)"
    >
    <span data-text="$result"></span>
</div>
```

```
function myfunction(data) {
    return `You entered: ${data}`;
}
```

If your function call is asynchronous then it will need to dispatch a custom event containing the result. While asynchronous code *can* be placed within Datastar expressions, Datastar will *not* await it.

```
<div data-signals:result>
    <input data-bind:foo 
           data-on:input="myfunction(el, $foo)"
           data-on:mycustomevent__window="$result = evt.detail.value"
    >
    <span data-text="$result"></span>
</div>
```

```
async function myfunction(element, data) {
    const value = await new Promise((resolve) => {
        setTimeout(() => resolve(`You entered: ${data}`), 1000);
    });
    element.dispatchEvent(
        new CustomEvent('mycustomevent', {detail: {value}})
    );
}
```

See the [sortable example](https://data-star.dev/examples/sortable).

### Web Components 

[Web components](https://developer.mozilla.org/en-US/docs/Web/API/Web_components) allow you to create reusable, encapsulated, custom elements. They are native to the web and require no external libraries or frameworks. Web components unlock [custom elements](https://developer.mozilla.org/en-US/docs/Web/API/Web_components/Using_custom_elements) – HTML tags with custom behavior and styling.

When using web components, pass data into them via attributes and listen for custom events dispatched from them (*props down, events up*).

In this way, the web component is encapsulated – all it knows is that it receives input via an attribute, acts on it, and optionally dispatches a custom event containing the result – and `data-*` attributes can be used to drive reactivity.

```
<div data-signals:result="''">
    <input data-bind:foo />
    <my-component
        data-attr:src="$foo"
        data-on:mycustomevent="$result = evt.detail.value"
    ></my-component>
    <span data-text="$result"></span>
</div>
```

```
class MyComponent extends HTMLElement {
    static get observedAttributes() {
        return ['src'];
    }

    attributeChangedCallback(name, oldValue, newValue) {
        const value = `You entered: ${newValue}`;
        this.dispatchEvent(
            new CustomEvent('mycustomevent', {detail: {value}})
        );
    }
}

customElements.define('my-component', MyComponent);
```

Since the `value` attribute is allowed on web components, it is also possible to use `data-bind` to bind a signal to the web component’s value. Note that a `change` event must be dispatched so that the event listener used by `data-bind` is triggered by the value change.

See the [web component example](https://data-star.dev/examples/web_component).

## Executing Scripts 

Just like elements and signals, the backend can also send JavaScript to be executed on the frontend using [backend actions](https://data-star.dev/reference/actions#backend-actions).

```
<button data-on:click="@get('/endpoint')">
    What are you talking about, HAL?
</button>
```

If a response has a `content-type` of `text/javascript`, the value will be executed as JavaScript in the browser.

```
alert('This mission is too important for me to allow you to jeopardize it.')
```

Demo

What are you talking about, HAL?

If the response has a `content-type` of `text/event-stream`, it can contain zero or more [SSE events](https://data-star.dev/reference/sse_events). The example above can be replicated by including a `script` tag inside of a `datastar-patch-elements` SSE event.

```
event: datastar-patch-elements
data: elements <div id="hal">
data: elements     <script>alert('This mission is too important for me to allow you to jeopardize it.')</script>
data: elements </div>

```

If you *only* want to execute a script, you can `append` the script tag to the `body`.

```
event: datastar-patch-elements
data: mode append
data: selector body
data: elements <script>alert('This mission is too important for me to allow you to jeopardize it.')</script>

```

Most SDKs have an `ExecuteScript` helper function for executing a script. Here’s the code to generate the SSE event above using the Go SDK.

```
sse := datastar.NewSSE(writer, request)
sse.ExecuteScript(`alert('This mission is too important for me to allow you to jeopardize it.')`)
```

Demo

What are you talking about, HAL?

We’ll cover event streams and [SSE events](https://data-star.dev/reference/sse_events) in more detail [later in the guide](https://data-star.dev/guide/backend_requests), but as you can see, they are just plain text events with a special syntax, made simpler by the [SDKs](https://data-star.dev/reference/sdks).

# Backend Requests

Between [attributes](https://data-star.dev/reference/attributes) and [actions](https://data-star.dev/reference/actions), Datastar provides you with everything you need to build hypermedia-driven applications. Using this approach, the backend drives state to the frontend and acts as the single source of truth, determining what actions the user can take next.

## Sending Signals 

By default, all signals (except for local signals whose keys begin with an underscore) are sent in an object with every backend request. When using a `GET` request, the signals are sent as a `datastar` query parameter, otherwise they are sent as a JSON body.

By sending **all** signals in every request, the backend has full access to the frontend state. This is by design. It is **not** recommended to send partial signals, but if you must, you can use the [`filterSignals`](https://data-star.dev/reference/actions#filterSignals) option to filter the signals sent to the backend.

### Nesting Signals 

Signals can be nested, making it easier to target signals in a more granular way on the backend.

Using dot-notation:

```
<div data-signals:foo.bar="1"></div>
```

Using object syntax:

```
<div data-signals="{foo: {bar: 1}}"></div>
```

Using two-way binding:

```
<input data-bind:foo.bar />
```

A practical use-case of nested signals is when you have repetition of state on a page. The following example tracks the open/closed state of a menu on both desktop and mobile devices, and the [toggleAll()](https://data-star.dev/reference/actions#toggleAll) action to toggle the state of all menus at once.

```
<div data-signals="{menu: {isOpen: {desktop: false, mobile: false}}}">
    <button data-on:click="@toggleAll({include: /^menu\.isOpen\./})">
        Open/close menu
    </button>
</div>
```

## Reading Signals 

To read signals from the backend, JSON decode the `datastar` query param for `GET` requests, and the request body for all other methods.

All [SDKs](https://data-star.dev/reference/sdks) provide a helper function to read signals. Here’s how you would read the nested signal `foo.bar` from an incoming request.


```
using StarFederation.Datastar.DependencyInjection;

// Adds Datastar as a service
builder.Services.AddDatastar();

public record Signals
{
    [JsonPropertyName("foo")] [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public FooSignals? Foo { get; set; } = null;

    public record FooSignals
    {
        [JsonPropertyName("bar")] [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        public string? Bar { get; set; }
    }
}

app.MapGet("/read-signals", async (IDatastarService datastarService) =>
{
    Signals? mySignals = await datastarService.ReadSignalsAsync<Signals>();
    var bar = mySignals?.Foo?.Bar;
});
```

```
import ("github.com/starfederation/datastar-go/datastar")

type Signals struct {
    Foo struct {
        Bar string `json:"bar"`
    } `json:"foo"`
}

signals := &Signals{}
if err := datastar.ReadSignals(request, signals); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
```


```
@Serializable
data class Signals(
    val foo: String,
)

val jsonUnmarshaller: JsonUnmarshaller<Signals> = { json -> Json.decodeFromString(json) }

val request: Request =
    postRequest(
        body =
            """
            {
                "foo": "bar"
            }
            """.trimIndent(),
    )

val signals = readSignals(request, jsonUnmarshaller)
```

```
use starfederation\datastar\ServerSentEventGenerator;

// Reads all signals from the request.
$signals = ServerSentEventGenerator::readSignals();
```

```
from datastar_py.fastapi import datastar_response, read_signals

@app.get("/updates")
@datastar_response
async def updates(request: Request):
    # Retrieve a dictionary with the current state of the signals from the frontend
    signals = await read_signals(request)
```

```
# Setup with request
datastar = Datastar.new(request:, response:)

# Read signals
some_signal = datastar.signals[:some_signal]
```



## SSE Events 

Datastar can stream zero or more [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) (SSE) from the web server to the browser. There’s no special backend plumbing required to use SSE, just some special syntax. Fortunately, SSE is straightforward and [provides us with some advantages](https://data-star.dev/essays/event_streams_all_the_way_down), in addition to allowing us to send multiple events in a single response (in contrast to sending `text/html` or `application/json` responses).

First, set up your backend in the language of your choice. Familiarize yourself with [sending SSE events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#sending_events_from_the_server), or use one of the backend [SDKs](https://data-star.dev/reference/sdks) to get up and running even faster. We’re going to use the SDKs in the examples below, which set the appropriate headers and format the events for us.

The following code would exist in a controller action endpoint in your backend.

```
;; Import the SDK's api and your adapter
(require
 '[starfederation.datastar.clojure.api :as d*]
 '[starfederation.datastar.clojure.adapter.http-kit :refer [->sse-response on-open]])

;; in a ring handler
(defn handler [request]
  ;; Create an SSE response
  (->sse-response request
                  {on-open
                   (fn [sse]
                     ;; Patches elements into the DOM
                     (d*/patch-elements! sse
                                         "<div id=\"question\">What do you put in a toaster?</div>")

                     ;; Patches signals
                     (d*/patch-signals! sse "{response: '', answer: 'bread'}"))}))
```

```
using StarFederation.Datastar.DependencyInjection;

// Adds Datastar as a service
builder.Services.AddDatastar();

app.MapGet("/", async (IDatastarService datastarService) =>
{
    // Patches elements into the DOM.
    await datastarService.PatchElementsAsync(@"<div id=""question"">What do you put in a toaster?</div>");

    // Patches signals.
    await datastarService.PatchSignalsAsync(new { response = "", answer = "bread" });
});
```

```
import ("github.com/starfederation/datastar-go/datastar")

// Creates a new `ServerSentEventGenerator` instance.
sse := datastar.NewSSE(w,r)

// Patches elements into the DOM.
sse.PatchElements(
    `<div id="question">What do you put in a toaster?</div>`
)

// Patches signals.
sse.PatchSignals([]byte(`{response: '', answer: 'bread'}`))
```

```
import starfederation.datastar.utils.ServerSentEventGenerator;

// Creates a new `ServerSentEventGenerator` instance.
AbstractResponseAdapter responseAdapter = new HttpServletResponseAdapter(response);
ServerSentEventGenerator generator = new ServerSentEventGenerator(responseAdapter);

// Patches elements into the DOM.
generator.send(PatchElements.builder()
    .data("<div id=\"question\">What do you put in a toaster?</div>")
    .build()
);

// Patches signals.
generator.send(PatchSignals.builder()
    .data("{\"response\": \"\", \"answer\": \"\"}")
    .build()
);
```

```
val generator = ServerSentEventGenerator(response)

generator.patchElements(
    elements = """<div id="question">What do you put in a toaster?</div>""",
)

generator.patchSignals(
    signals = """{"response": "", "answer": "bread"}""",
)
```

```
use starfederation\datastar\ServerSentEventGenerator;

// Creates a new `ServerSentEventGenerator` instance.
$sse = new ServerSentEventGenerator();

// Patches elements into the DOM.
$sse->patchElements(
    '<div id="question">What do you put in a toaster?</div>'
);

// Patches signals.
$sse->patchSignals(['response' => '', 'answer' => 'bread']);
```

```
from datastar_py import ServerSentEventGenerator as SSE
from datastar_py.litestar import DatastarResponse

async def endpoint():
    return DatastarResponse([
        SSE.patch_elements('<div id="question">What do you put in a toaster?</div>'),
        SSE.patch_signals({"response": "", "answer": "bread"})
    ])
```

```
require 'datastar'

# Create a Datastar::Dispatcher instance

datastar = Datastar.new(request:, response:)

# In a Rack handler, you can instantiate from the Rack env
# datastar = Datastar.from_rack_env(env)

# Start a streaming response
datastar.stream do |sse|
  # Patches elements into the DOM
  sse.patch_elements %(<div id="question">What do you put in a toaster?</div>)

  # Patches signals
  sse.patch_signals(response: '', answer: 'bread')
end
```

```
use datastar::prelude::*;
use async_stream::stream;

Sse(stream! {
    // Patches elements into the DOM.
    yield PatchElements::new("<div id='question'>What do you put in a toaster?</div>").into();

    // Patches signals.
    yield PatchSignals::new("{response: '', answer: 'bread'}").into();
})
```

```
// Creates a new `ServerSentEventGenerator` instance (this also sends required headers)
ServerSentEventGenerator.stream(req, res, (stream) => {
      // Patches elements into the DOM.
     stream.patchElements(`<div id="question">What do you put in a toaster?</div>`);

     // Patches signals.
     stream.patchSignals({'response':  '', 'answer': 'bread'});
});
```

The `PatchElements()` function updates the provided HTML element into the DOM, replacing the element with `id="question"`. An element with the ID `question` must *already* exist in the DOM.

The `PatchSignals()` function updates the `response` and `answer` signals into the frontend signals.

With our backend in place, we can now use the `data-on:click` attribute to trigger the [`@get()`](https://data-star.dev/reference/actions#get) action, which sends a `GET` request to the `/actions/quiz` endpoint on the server when a button is clicked.

```
<div
    data-signals="{response: '', answer: ''}"
    data-computed:correct="$response.toLowerCase() == $answer"
>
    <div id="question"></div>
    <button data-on:click="@get('/actions/quiz')">Fetch a question</button>
    <button
        data-show="$answer != ''"
        data-on:click="$response = prompt('Answer:') ?? ''"
    >
        BUZZ
    </button>
    <div data-show="$response != ''">
        You answered “<span data-text="$response"></span>”.
        <span data-show="$correct">That is correct ✅</span>
        <span data-show="!$correct">
        The correct answer is “<span data-text="$answer"></span>” 🤷
        </span>
    </div>
</div>
```

Now when the `Fetch a question` button is clicked, the server will respond with an event to modify the `question` element in the DOM and an event to modify the `response` and `answer` signals. We’re driving state from the backend!

Demo

...

Fetch a question BUZZ

You answered “”. That is correct ✅ The correct answer is “” 🤷

### `data-indicator` 

The [`data-indicator`](https://data-star.dev/reference/attributes#data-indicator) attribute sets the value of a signal to `true` while the request is in flight, otherwise `false`. We can use this signal to show a loading indicator, which may be desirable for slower responses.

```
<div id="question"></div>
<button
    data-on:click="@get('/actions/quiz')"
    data-indicator:fetching
>
    Fetch a question
</button>
<div data-class:loading="$fetching" class="indicator"></div>
```

Demo

...

Fetch a question

![Indicator](https://data-star.dev/cdn-cgi/image/format=auto,width=32/static/images/rocket-animated-1d781383a0d7cbb1eb575806abeec107c8a915806fb55ee19e4e33e8632c75e5.gif)

## Backend Actions 

We’re not limited to sending just `GET` requests. Datastar provides [backend actions](https://data-star.dev/reference/actions#backend-actions) for each of the methods available: `@get()`, `@post()`, `@put()`, `@patch()` and `@delete()`.

Here’s how we can send an answer to the server for processing, using a `POST` request.

```
<button data-on:click="@post('/actions/quiz')">
    Submit answer
</button>
```

One of the benefits of using SSE is that we can send multiple events (patch elements and patch signals) in a single response.

```
(d*/patch-elements! sse "<div id=\"question\">...</div>")
(d*/patch-elements! sse "<div id=\"instructions\">...</div>")
(d*/patch-signals! sse "{answer: '...', prize: '...'}")
```

```
datastarService.PatchElementsAsync(@"<div id=""question"">...</div>");
datastarService.PatchElementsAsync(@"<div id=""instructions"">...</div>");
datastarService.PatchSignalsAsync(new { answer = "...", prize = "..." } );
```

```
sse.PatchElements(`<div id="question">...</div>`)
sse.PatchElements(`<div id="instructions">...</div>`)
sse.PatchSignals([]byte(`{answer: '...', prize: '...'}`))
```

```
generator.send(PatchElements.builder()
    .data("<div id=\"question\">...</div>")
    .build()
);
generator.send(PatchElements.builder()
    .data("<div id=\"instructions\">...</div>")
    .build()
);
generator.send(PatchSignals.builder()
    .data("{\"answer\": \"...\", \"prize\": \"...\"}")
    .build()
);
```

```
generator.patchElements(
    elements = """<div id="question">...</div>""",
)
generator.patchElements(
    elements = """<div id="instructions">...</div>""",
)
generator.patchSignals(
    signals = """{"answer": "...", "prize": "..."}""",
)
```

```
$sse->patchElements('<div id="question">...</div>');
$sse->patchElements('<div id="instructions">...</div>');
$sse->patchSignals(['answer' => '...', 'prize' => '...']);
```

```
return DatastarResponse([
    SSE.patch_elements('<div id="question">...</div>'),
    SSE.patch_elements('<div id="instructions">...</div>'),
    SSE.patch_signals({"answer": "...", "prize": "..."})
])
```

```
datastar.stream do |sse|
  sse.patch_elements('<div id="question">...</div>')
  sse.patch_elements('<div id="instructions">...</div>')
  sse.patch_signals(answer: '...', prize: '...')
end
```

```
yield PatchElements::new("<div id='question'>...</div>").into()
yield PatchElements::new("<div id='instructions'>...</div>").into()
yield PatchSignals::new("{answer: '...', prize: '...'}").into()
```

```
stream.patchElements('<div id="question">...</div>');
stream.patchElements('<div id="instructions">...</div>');
stream.patchSignals({'answer': '...', 'prize': '...'});
```

> In addition to your browser’s dev tools, the [Datastar Inspector](https://data-star.dev/pro#datastar-inspector) can be used to monitor and inspect SSE events received by Datastar.

Read more about SSE events in the [reference](https://data-star.dev/reference/sse_events).

## Congratulations 

You’ve actually read the entire guide! You should now know how to use Datastar to build reactive applications that communicate with the backend using backend requests and SSE events.

Feel free to dive into the [reference](https://data-star.dev/reference) and explore the [examples](https://data-star.dev/examples) next, to learn more about what you can do with Datastar.

If you’re wondering how to best use Datastar to build maintainable, scalable, high-performance web apps, read (and re-read) the [Tao of Datastar](https://data-star.dev/guide/the_tao_of_datastar).

![The Tao of Datastar](https://data-star.dev/cdn-cgi/image/format=auto,width=640/static/images/tao-of-datastar-454a92131f2d9d30fb17c6e1c86b56833717cc5e25c318738cfa225b0b3c69f0.png)

# The Tao of Datastar

Datastar is just a tool. The Tao of Datastar, or “the Datastar way” as it is often referred to, is a set of opinions from the core team on how to best use Datastar to build maintainable, scalable, high-performance web apps.

Ignore them at your own peril!

![The Tao of Datastar](https://data-star.dev/cdn-cgi/image/format=auto,width=640/static/images/tao-of-datastar-454a92131f2d9d30fb17c6e1c86b56833717cc5e25c318738cfa225b0b3c69f0.png)

## State in the Right Place 

Most state should live in the backend. Since the frontend is exposed to the user, the backend should be the source of truth for your application state.

## Start with the Defaults 

The default configuration options are the recommended settings for the majority of applications. Start with the defaults, and before you ever get tempted to change them, stop and ask yourself, [well... how did I get here?](https://youtu.be/5IsSpAOD6K8)

## Patch Elements & Signals 

Since the backend is the source of truth, it should *drive* the frontend by **patching** (adding, updating and removing) HTML elements and signals.

## Use Signals Sparingly 

Overusing signals typically indicates trying to manage state on the frontend. Favor fetching current state from the backend rather than pre-loading and assuming frontend state is current. A good rule of thumb is to *only* use signals for user interactions (e.g. toggling element visibility) and for sending new state to the backend (e.g. by binding signals to form input elements).

## In Morph We Trust 

Morphing ensures that only modified parts of the DOM are updated, preserving state and improving performance. This allows you to send down large chunks of the DOM tree (all the way up to the `html` tag), sometimes known as “fat morph”, rather than trying to manage fine-grained updates yourself. If you want to explicitly ignore morphing an element, place the [`data-ignore-morph`](https://data-star.dev/reference/attributes#data-ignore-morph) attribute on it.

## SSE Responses 

[SSE](https://html.spec.whatwg.org/multipage/server-sent-events.html) responses allow you to send `0` to `n` events, in which you can [patch elements](https://data-star.dev/guide/getting_started/#patching-elements), [patch signals](https://data-star.dev/guide/reactive_signals#patching-signals), and [execute scripts](https://data-star.dev/guide/datastar_expressions#executing-scripts). Since event streams are just HTTP responses with some special formatting that [SDKs](https://data-star.dev/reference/sdks) can handle for you, there’s no real benefit to using a content type other than [`text/event-stream`](https://data-star.dev/reference/actions#response-handling).

## Compression 

Since SSE responses stream events from the backend and morphing allows sending large chunks of DOM, compressing the response is a natural choice. Compression ratios of 200:1 are not uncommon when compressing streams using Brotli. Read more about compressing streams in [this article](https://andersmurphy.com/2025/04/15/why-you-should-use-brotli-sse.html).

## Backend Templating 

Since your backend generates your HTML, you can and should use your templating language to [keep things DRY](https://data-star.dev/how_tos/keep_datastar_code_dry) (Don’t Repeat Yourself).

## Page Navigation 

Page navigation hasn't changed in 30 years. Use the [anchor element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/a) (`<a>`) to navigate to a new page, or a [redirect](https://data-star.dev/how_tos/redirect_the_page_from_the_backend) if redirecting from the backend. For smooth page transitions, use the [View Transition API](https://developer.mozilla.org/en-US/docs/Web/API/View_Transition_API).

## Browser History 

Browsers automatically keep a history of pages visited. As soon as you start trying to manage browser history yourself, you are adding complexity. Each page is a resource. Use anchor tags and let the browser do what it is good at.

## CQRS 

[CQRS](https://martinfowler.com/bliki/CQRS.html), in which commands (writes) and requests (reads) are segregated, makes it possible to have a single long-lived request to receive updates from the backend (reads), while making multiple short-lived requests to the backend (writes). It is a powerful pattern that makes real-time collaboration simple using Datastar. Here’s a basic example.

```
<div id="main" data-init="@get('/cqrs_endpoint')">
    <button data-on:click="@post('/do_something')">
        Do something
    </button>
</div>
```

## Loading Indicators 

Loading indicators inform the user that an action is in progress. Use the [`data-indicator`](https://data-star.dev/reference/attributes#data-indicator) attribute to show loading indicators on elements that trigger backend requests. Here’s an example of a button that shows a loading element while waiting for a response from the backend.

```
<div>
    <button data-indicator:_loading
            data-on:click="@post('/do_something')"
    >
        Do something
        <span data-show="$_loading">Loading...</span>
    </button>
</div>
```

When using [CQRS](#cqrs), it is generally better to manually show a loading indicator when backend requests are made, and allow it to be hidden when the DOM is updated from the backend. Here’s an example.

```
<div>
    <button data-on:click="el.classList.add('loading'); @post('/do_something')">
        Do something
        <span>Loading...</span>
    </button>
</div>
```

## Optimistic Updates 

Optimistic updates (also known as optimistic UI) are when the UI updates immediately as if an operation succeeded, before the backend actually confirms it. It is a strategy used to makes web apps feel snappier, when it in fact deceives the user. Imagine seeing a confirmation message that an action succeeded, only to be shown a second later that it actually failed. Rather than deceive the user, use [loading indicators](#loading-indicators) to show the user that the action is in progress, and only confirm success from the backend (see [this example](https://data-star.dev/examples/rocket_flow)).

## Accessibility 

The web should be accessible to everyone. Datastar stays out of your way and leaves [accessibility](https://developer.mozilla.org/en-US/docs/Web/Accessibility) to you. Use semantic HTML, apply ARIA where it makes sense, and ensure your app works well with keyboards and screen readers. Here’s an example of using [`data-attr`](https://data-star.dev/reference/attributes#data-attr) to apply ARIA attributes to a button that toggles the visibility of a menu.

```
<button data-on:click="$_menuOpen = !$_menuOpen"
        data-attr:aria-expanded="$_menuOpen ? 'true' : 'false'"
>
    Open/Close Menu
</button>
<div data-attr:aria-hidden="$_menuOpen ? 'false' : 'true'"></div>
```

# Attributes

Data attributes are [evaluated in the order](#attribute-evaluation-order) they appear in the DOM, have special [casing](#attribute-casing) rules, can be [aliased](#aliasing-attributes) to avoid conflicts with other libraries, can contain [Datastar expressions](#datastar-expressions), and have [runtime error handling](#error-handling).

> The Datastar [VSCode extension](https://marketplace.visualstudio.com/items?itemName=starfederation.datastar-vscode) and [IntelliJ plugin](https://plugins.jetbrains.com/plugin/26072-datastar-support) provide autocompletion for all available `data-*` attributes.

### `data-attr` 

Sets the value of any HTML attribute to an expression, and keeps it in sync.

```
<div data-attr:aria-label="$foo"></div>
```

The `data-attr` attribute can also be used to set the values of multiple attributes on an element using a set of key-value pairs, where the keys represent attribute names and the values represent expressions.

```
<div data-attr="{'aria-label': $foo, disabled: $bar}"></div>
```

### `data-bind` 

Creates a signal (if one doesn’t already exist) and sets up two-way data binding between it and an element’s current bound state. When the signal changes, Datastar writes that value to the element. When one of the bind events fires, Datastar reads the element’s current bound property/value and writes that back to the signal.

The `data-bind` attribute can be placed on any HTML element on which data can be input or choices selected (`input`, `select`, `textarea` elements, and web components). Native elements use their built-in bind semantics automatically. Generic custom elements default to binding through `value` and listening on `change`.

`data-bind` does **not** inspect the event payload. It only uses the configured event as a signal to re-read the element’s current bound property/value. If you need to pull data from `event` itself, use `data-on:*` instead.

```
<input data-bind:foo />
```

The signal name can be specified in the key (as above), or in the value (as below). This can be useful depending on the templating language you are using.

```
<input data-bind="foo" />
```

[Attribute casing](#attribute-casing) rules apply to the signal name.

```
<!-- Both of these create the signal `$fooBar` -->
<input data-bind:foo-bar />
<input data-bind="fooBar" />
```

The initial value of the signal is set to the value of the element, unless a signal has already been defined. So in the example below, `$fooBar` is set to `baz`.

```
<input data-bind:foo-bar value="baz" />
```

Whereas in the example below, `$fooBar` inherits the value `fizz` of the predefined signal.

```
<div data-signals:foo-bar="'fizz'">
    <input data-bind:foo-bar value="baz" />
</div>
```

#### Predefined Signal Types

When you predefine a signal, its **type** is preserved during binding. Whenever the element’s value changes, the signal value is automatically converted to match the original type.

For example, in the code below, `$fooBar` is set to the **number** `10` (not the string `"10"`) when the option is selected.

```
<div data-signals:foo-bar="0">
    <select data-bind:foo-bar>
        <option value="10">10</option>
    </select>
</div>
```

In the same way, you can assign multiple input values to a single signal by predefining it as an **array**. In the example below, `$fooBar` becomes `["fizz", "baz"]` when both checkboxes are checked, and `["", ""]` when neither is checked.

```
<div data-signals:foo-bar="[]">
    <input data-bind:foo-bar type="checkbox" value="fizz" />
    <input data-bind:foo-bar type="checkbox" value="baz" />
</div>
```

#### File Uploads

Input fields of type `file` will automatically encode file contents in base64. This means that a form is not required.

```
<input type="file" data-bind:files multiple />
```

The resulting signal is in the format `{ name: string, contents: string, mime: string }[]`. See the [file upload example](https://data-star.dev/examples/file_upload).

> If you want files to be uploaded to the server, rather than be converted to signals, use a form and with `multipart/form-data` in the [`enctype`](https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement/enctype) attribute. See the [backend actions](https://data-star.dev/reference/actions#backend-actions) reference.

#### Modifiers

Modifiers allow you to modify behavior when binding signals using a key.

- `__case` – Converts the casing of the signal name.
  
  - `.camel` – Camel case: `mySignal` (default)
  - `.kebab` – Kebab case: `my-signal`
  - `.snake` – Snake case: `my_signal`
  - `.pascal` – Pascal case: `MySignal`
- `__prop` – Binds to a specific property instead of the default binding. Must *not* be a read-only property.
  
  - Example: `data-bind:is-checked__prop.checked`
- `__event` – Defines which events sync the element property back to the signal.
  
  - Example: `data-bind:query__event.input.change`

Native form controls use their built-in binding semantics automatically. Generic custom elements default to `value` and `change`. Use `__prop` and `__event` when a custom element’s live state is stored somewhere else.

```
<my-toggle data-bind:is-checked__prop.checked__event.change></my-toggle>
```

### `data-class` 

Adds or removes a class to or from an element based on an expression.

```
<div data-class:font-bold="$foo == 'strong'"></div>
```

If the expression evaluates to `true`, the `hidden` class is added to the element; otherwise, it is removed.

The `data-class` attribute can also be used to add or remove multiple classes from an element using a set of key-value pairs, where the keys represent class names and the values represent expressions.

```
<div data-class="{success: $foo != '', 'font-bold': $foo == 'strong'}"></div>
```

#### Modifiers

Modifiers allow you to modify behavior when defining a class name using a key.

- `__case` – Converts the casing of the class.
  
  - `.camel` – Camel case: `myClass`
  - `.kebab` – Kebab case: `my-class` (default)
  - `.snake` – Snake case: `my_class`
  - `.pascal` – Pascal case: `MyClass`

```
<div data-class:my-class__case.camel="$foo"></div>
```

### `data-computed` 

Creates a signal that is computed based on an expression. The computed signal is read-only, and its value is automatically updated when any signals in the expression are updated.

```
<div data-computed:foo="$bar + $baz"></div>
```

Computed signals are useful for memoizing expressions containing other signals. Their values can be used in other expressions.

```
<div data-computed:foo="$bar + $baz"></div>
<div data-text="$foo"></div>
```

> Computed signal expressions must not be used for performing actions (changing other signals, actions, JavaScript functions, etc.). If you need to perform an action in response to a signal change, use the [`data-effect`](#data-effect) attribute.

The `data-computed` attribute can also be used to create computed signals using a set of key-value pairs, where the keys represent signal names and the values are callables (usually arrow functions) that return a reactive value.

```
<div data-computed="{foo: () => $bar + $baz}"></div>
```

#### Modifiers

Modifiers allow you to modify behavior when defining computed signals using a key.

- `__case` – Converts the casing of the signal name.
  
  - `.camel` – Camel case: `mySignal` (default)
  - `.kebab` – Kebab case: `my-signal`
  - `.snake` – Snake case: `my_signal`
  - `.pascal` – Pascal case: `MySignal`

```
<div data-computed:my-signal__case.kebab="$bar + $baz"></div>
```

### `data-effect` 

Executes an expression on page load and whenever any signals in the expression change. This is useful for performing side effects, such as updating other signals, making requests to the backend, or manipulating the DOM.

```
<div data-effect="$foo = $bar + $baz"></div>
```

### `data-ignore` 

Datastar walks the entire DOM and applies plugins to each element it encounters. It’s possible to tell Datastar to ignore an element and its descendants by placing a `data-ignore` attribute on it. This can be useful for preventing naming conflicts with third-party libraries, or when you are unable to [escape user input](https://data-star.dev/reference/security#escape-user-input).

```
<div data-ignore data-show-thirdpartylib="">
    <div>
        Datastar will not process this element.
    </div>
</div>
```

#### Modifiers

- `__self` – Only ignore the element itself, not its descendants.

### `data-ignore-morph` 

Similar to the `data-ignore` attribute, the `data-ignore-morph` attribute tells the `PatchElements` watcher to skip processing an element and its children when morphing elements.

```
<div data-ignore-morph>
    This element will not be morphed.
</div>
```

> To remove the `data-ignore-morph` attribute from an element, simply patch the element with the `data-ignore-morph` attribute removed.

### `data-indicator` 

Creates a signal and sets its value to `true` while a fetch request is in flight, otherwise `false`. The signal can be used to show a loading indicator.

```
<button data-on:click="@get('/endpoint')"
        data-indicator:fetching
></button>
```

This can be useful for showing a loading spinner, disabling a button, etc.

```
<button data-on:click="@get('/endpoint')"
        data-indicator:fetching
        data-attr:disabled="$fetching"
></button>
<div data-show="$fetching">Loading...</div>
```

The signal name can be specified in the key (as above), or in the value (as below). This can be useful depending on the templating language you are using.

```
<button data-indicator="fetching"></button>
```

When using `data-indicator` with a fetch request initiated in a `data-init` attribute, you should ensure that the indicator signal is created before the fetch request is initialized.

```
<div data-indicator:fetching data-init="@get('/endpoint')"></div>
```

#### Modifiers

Modifiers allow you to modify behavior when defining indicator signals using a key.

- `__case` – Converts the casing of the signal name.
  
  - `.camel` – Camel case: `mySignal` (default)
  - `.kebab` – Kebab case: `my-signal`
  - `.snake` – Snake case: `my_signal`
  - `.pascal` – Pascal case: `MySignal`

### `data-init` 

Runs an expression when the attribute is initialized. This can happen on page load, when an element is patched into the DOM, and any time the attribute is modified (via a backend action or otherwise).

> The expression contained in the [`data-init`](#data-init) attribute is executed when the element attribute is loaded into the DOM. This can happen on page load, when an element is patched into the DOM, and any time the attribute is modified (via a backend action or otherwise).

```
<div data-init="$count = 1"></div>
```

#### Modifiers

Modifiers allow you to add a delay to the event listener.

- `__delay` – Delay the event listener.
  
  - `.500ms` – Delay for 500 milliseconds (accepts any integer).
  - `.1s` – Delay for 1 second (accepts any integer).
- `__viewtransition` – Wraps the expression in `document.startViewTransition()` when the View Transition API is available.

```
<div data-init__delay.500ms="$count = 1"></div>
```

### `data-json-signals` 

Sets the text content of an element to a reactive JSON stringified version of signals. Useful when troubleshooting an issue.

```
<!-- Display all signals -->
<pre data-json-signals></pre>
```

You can optionally provide a filter object to include or exclude specific signals using regular expressions.

```
<!-- Only show signals that include "user" in their path -->
<pre data-json-signals="{include: /user/}"></pre>

<!-- Show all signals except those ending in "temp" -->
<pre data-json-signals="{exclude: /temp$/}"></pre>

<!-- Combine include and exclude filters -->
<pre data-json-signals="{include: /^app/, exclude: /password/}"></pre>
```

#### Modifiers

Modifiers allow you to modify the output format.

- `__terse` – Outputs a more compact JSON format without extra whitespace. Useful for displaying filtered data inline.

```
<!-- Display filtered signals in a compact format -->
<pre data-json-signals__terse="{include: /counter/}"></pre>
```

### `data-on` 

Attaches an event listener to an element, executing an expression whenever the event is triggered.

```
<button data-on:click="$foo = ''">Reset</button>
```

An `evt` variable that represents the event object is available in the expression.

```
<div data-on:my-event="$foo = evt.detail"></div>
```

The `data-on` attribute works with [events](https://developer.mozilla.org/en-US/docs/Web/Events) and [custom events](https://developer.mozilla.org/en-US/docs/Web/Events/Creating_and_triggering_events). The `data-on:submit` event listener prevents the default submission behavior of forms.

#### Modifiers

Modifiers allow you to modify behavior when events are triggered. Some modifiers have tags to further modify the behavior.

- `__once` * – Only trigger the event listener once.
- `__passive` * – Do not call `preventDefault` on the event listener.
- `__capture` * – Use a capture event listener.
- `__case` – Converts the casing of the event.
  
  - `.camel` – Camel case: `myEvent`
  - `.kebab` – Kebab case: `my-event` (default)
  - `.snake` – Snake case: `my_event`
  - `.pascal` – Pascal case: `MyEvent`
- `__delay` – Delay the event listener.
  
  - `.500ms` – Delay for 500 milliseconds (accepts any integer).
  - `.1s` – Delay for 1 second (accepts any integer).
- `__debounce` – Debounce the event listener.
  
  - `.500ms` – Debounce for 500 milliseconds (accepts any integer).
  - `.1s` – Debounce for 1 second (accepts any integer).
  - `.leading` – Debounce with leading edge (must come after timing).
  - `.notrailing` – Debounce without trailing edge (must come after timing).
- `__throttle` – Throttle the event listener.
  
  - `.500ms` – Throttle for 500 milliseconds (accepts any integer).
  - `.1s` – Throttle for 1 second (accepts any integer).
  - `.noleading` – Throttle without leading edge (must come after timing).
  - `.trailing` – Throttle with trailing edge (must come after timing).
- `__viewtransition` – Wraps the expression in `document.startViewTransition()` when the View Transition API is available.
- `__window` – Attaches the event listener to the `window` element.
- `__document` – Attaches the event listener to the `document` element. Useful for events that are only available on `document` and that do not bubble.
- `__outside` – Triggers when the event is outside the element.
- `__prevent` – Calls `preventDefault` on the event listener.
- `__stop` – Calls `stopPropagation` on the event listener.

** Only works with built-in events.*

```
<button data-on:click__window__debounce.500ms.leading="$foo = ''"></button>
<div data-on:my-event__case.camel="$foo = ''"></div>
```

### `data-on-intersect` 

Runs an expression when the element intersects with the viewport.

```
<div data-on-intersect="$intersected = true"></div>
```

#### Modifiers

Modifiers allow you to modify the element intersection behavior and the timing of the event listener.

- `__once` – Only triggers the event once.
- `__exit` – Only triggers the event when the element exits the viewport.
- `__half` – Triggers when half of the element is visible.
- `__full` – Triggers when the full element is visible.
- `__threshold` – Triggers when the element is visible by a certain percentage.
  
  - `.25` – Triggers when 25% of the element is visible.
  - `.75` – Triggers when 75% of the element is visible.
- `__delay` – Delay the event listener.
  
  - `.500ms` – Delay for 500 milliseconds (accepts any integer).
  - `.1s` – Delay for 1 second (accepts any integer).
- `__debounce` – Debounce the event listener.
  
  - `.500ms` – Debounce for 500 milliseconds (accepts any integer).
  - `.1s` – Debounce for 1 second (accepts any integer).
  - `.leading` – Debounce with leading edge (must come after timing).
  - `.notrailing` – Debounce without trailing edge (must come after timing).
- `__throttle` – Throttle the event listener.
  
  - `.500ms` – Throttle for 500 milliseconds (accepts any integer).
  - `.1s` – Throttle for 1 second (accepts any integer).
  - `.noleading` – Throttle without leading edge (must come after timing).
  - `.trailing` – Throttle with trailing edge (must come after timing).
- `__viewtransition` – Wraps the expression in `document.startViewTransition()` when the View Transition API is available.

```
<div data-on-intersect__once__full="$fullyIntersected = true"></div>
```

### `data-on-interval` 

Runs an expression at a regular interval. The interval duration defaults to one second and can be modified using the `__duration` modifier.

```
<div data-on-interval="$count++"></div>
```

#### Modifiers

Modifiers allow you to modify the interval duration.

- `__duration` – Sets the interval duration.
  
  - `.500ms` – Interval duration of 500 milliseconds (accepts any integer).
  - `.1s` – Interval duration of 1 second (default).
  - `.leading` – Execute the first interval immediately.
- `__viewtransition` – Wraps the expression in `document.startViewTransition()` when the View Transition API is available.

```
<div data-on-interval__duration.500ms="$count++"></div>
```

### `data-on-signal-patch` 

Runs an expression whenever any signals are patched. This is useful for tracking changes, updating computed values, or triggering side effects when data updates.

```
<div data-on-signal-patch="console.log('A signal changed!')"></div>
```

The `patch` variable is available in the expression and contains the signal patch details.

```
<div data-on-signal-patch="console.log('Signal patch:', patch)"></div>
```

You can filter which signals to watch using the [`data-on-signal-patch-filter`](#data-on-signal-patch-filter) attribute.

#### Modifiers

Modifiers allow you to modify the timing of the event listener.

- `__delay` – Delay the event listener.
  
  - `.500ms` – Delay for 500 milliseconds (accepts any integer).
  - `.1s` – Delay for 1 second (accepts any integer).
- `__debounce` – Debounce the event listener.
  
  - `.500ms` – Debounce for 500 milliseconds (accepts any integer).
  - `.1s` – Debounce for 1 second (accepts any integer).
  - `.leading` – Debounce with leading edge (must come after timing).
  - `.notrailing` – Debounce without trailing edge (must come after timing).
- `__throttle` – Throttle the event listener.
  
  - `.500ms` – Throttle for 500 milliseconds (accepts any integer).
  - `.1s` – Throttle for 1 second (accepts any integer).
  - `.noleading` – Throttle without leading edge (must come after timing).
  - `.trailing` – Throttle with trailing edge (must come after timing).

```
<div data-on-signal-patch__debounce.500ms="doSomething()"></div>
```

### `data-on-signal-patch-filter` 

Filters which signals to watch when using the [`data-on-signal-patch`](#data-on-signal-patch) attribute.

The `data-on-signal-patch-filter` attribute accepts an object with `include` and/or `exclude` properties that are regular expressions.

```
<!-- Only react to counter signal changes -->
<div data-on-signal-patch-filter="{include: /^counter$/}"></div>

<!-- React to all changes except those ending with "changes" -->
<div data-on-signal-patch-filter="{exclude: /changes$/}"></div>

<!-- Combine include and exclude filters -->
<div data-on-signal-patch-filter="{include: /user/, exclude: /password/}"></div>
```

### `data-preserve-attr` 

Preserves the value of an attribute when morphing DOM elements.

```
<details open data-preserve-attr="open">
    <summary>Title</summary>
    Content
</details>
```

You can preserve multiple attributes by separating them with a space.

```
<details open class="foo" data-preserve-attr="open class">
    <summary>Title</summary>
    Content
</details>
```

### `data-ref` 

Creates a new signal that is a reference to the element on which the data attribute is placed.

```
<div data-ref:foo></div>
```

The signal name can be specified in the key (as above), or in the value (as below). This can be useful depending on the templating language you are using.

```
<div data-ref="foo"></div>
```

The signal value can then be used to reference the element.

```
$foo is a reference to a <span data-text="$foo.tagName"></span> element
```

#### Modifiers

Modifiers allow you to modify behavior when defining references using a key.

- `__case` – Converts the casing of the signal name.
  
  - `.camel` – Camel case: `mySignal` (default)
  - `.kebab` – Kebab case: `my-signal`
  - `.snake` – Snake case: `my_signal`
  - `.pascal` – Pascal case: `MySignal`

```
<div data-ref:my-signal__case.kebab></div>
```

### `data-show` 

Shows or hides an element based on whether an expression evaluates to `true` or `false`. For anything with custom requirements, use [`data-class`](#data-class) instead.

```
<div data-show="$foo"></div>
```

To prevent flickering of the element before Datastar has processed the DOM, you can add a `display: none` style to the element to hide it initially.

```
<div data-show="$foo" style="display: none"></div>
```

### `data-signals` 

Patches (adds, updates or removes) one or more signals into the existing signals. Values defined later in the DOM tree override those defined earlier.

```
<div data-signals:foo="1"></div>
```

Signals can be nested using dot-notation.

```
<div data-signals:foo.bar="1"></div>
```

The `data-signals` attribute can also be used to patch multiple signals using a set of key-value pairs, where the keys represent signal names and the values represent expressions.

```
<div data-signals="{foo: {bar: 1, baz: 2}}"></div>
```

The value above is written in JavaScript object notation, but JSON, which is a subset and which most templating languages have built-in support for, is also allowed.

Setting a signal’s value to `null` or `undefined` removes the signal.

```
<div data-signals="{foo: null}"></div>
```

Keys used in `data-signals:*` are converted to camel case, so the signal name `mySignal` must be written as `data-signals:my-signal` or `data-signals="{mySignal: 1}"`.

Signals beginning with an underscore are *not* included in requests to the backend by default. You can opt to include them by modifying the value of the [`filterSignals`](https://data-star.dev/reference/actions#filterSignals) option.

> Signal names cannot begin with nor contain a double underscore (`__`), due to its use as a modifier delimiter.

#### Modifiers

Modifiers allow you to modify behavior when patching signals using a key.

- `__case` – Converts the casing of the signal name.
  
  - `.camel` – Camel case: `mySignal` (default)
  - `.kebab` – Kebab case: `my-signal`
  - `.snake` – Snake case: `my_signal`
  - `.pascal` – Pascal case: `MySignal`
- `__ifmissing` – Only patches signals if their keys do not already exist. This is useful for setting defaults without overwriting existing values.

```
<div data-signals:my-signal__case.kebab="1"
     data-signals:foo__ifmissing="1"
></div>
```

### `data-style` 

Sets the value of inline CSS styles on an element based on an expression, and keeps them in sync.

```
<div data-style:display="$hiding && 'none'"></div>
<div data-style:background-color="$red ? 'red' : 'blue'"></div>
```

The `data-style` attribute can also be used to set multiple style properties on an element using a set of key-value pairs, where the keys represent CSS property names and the values represent expressions.

```
<div data-style="{
    display: $hiding ? 'none' : 'flex',
    'background-color': $red ? 'red' : 'green'
}"></div>
```

Empty string, `null`, `undefined`, or `false` values will restore the original inline style value if one existed, or remove the style property if there was no initial value. This allows you to use the logical AND operator (`&&`) for conditional styles: `$condition && 'value'` will apply the style when the condition is true and restore the original value when false.

```
<!-- When $x is false, color remains red from inline style -->
<div style="color: red;" data-style:color="$x && 'green'"></div>

<!-- When $hiding is true, display becomes none; when false, reverts to flex from inline style -->
<div style="display: flex;" data-style:display="$hiding && 'none'"></div>
```

The plugin tracks initial inline style values and restores them when data-style expressions become falsy or during cleanup. This ensures existing inline styles are preserved and only the dynamic changes are managed by Datastar.

### `data-text` 

Binds the text content of an element to an expression.

```
<div data-text="$foo"></div>
```

## Pro Attributes 

The Pro attributes add functionality to the free open source Datastar framework. These attributes are available under a [commercial license](https://data-star.dev/pro#license) that helps fund our open source work.

### `data-animate` [Pro](https://data-star.dev/pro "Datastar Pro")

Allows you to animate element attributes over time. Animated attributes are updated reactively whenever signals used in the expression change.

### `data-custom-validity` [Pro](https://data-star.dev/pro "Datastar Pro")

Allows you to add custom validity to an element using an expression. The expression must evaluate to a string that will be set as the custom validity message. If the string is empty, the input is considered valid. If the string is non-empty, the input is considered invalid and the string is used as the reported message.

```
<form>
    <input data-bind:foo name="foo" />
    <input data-bind:bar name="bar"
           data-custom-validity="$foo === $bar ? '' : 'Values must be the same.'"
    />
    <button>Submit form</button>
</form>
```

### `data-match-media` [Pro](https://data-star.dev/pro "Datastar Pro")

Sets a signal to whether a media query matches and keeps it in sync whenever the query changes.

```
<div
    data-match-media:is-dark="'prefers-color-scheme: dark'"
    data-computed:theme="$isDark ? 'dark' : 'light'"
></div>
```

The query value can be written as `prefers-color-scheme: dark` or `(prefers-color-scheme: dark)`, with or without surrounding quotes.

For more complex queries, pass a quoted query string with explicit media-query syntax (including parentheses) exactly as you want it evaluated by `window.matchMedia`.

See the [match media example](https://data-star.dev/examples/match_media).

#### Modifiers

Modifiers allow you to modify behavior when defining signals using a key.

- `__case` – Converts the casing of the signal name.
  
  - `.camel` – Camel case: `mySignal` (default)
  - `.kebab` – Kebab case: `my-signal`
  - `.snake` – Snake case: `my_signal`
  - `.pascal` – Pascal case: `MySignal`

```
<div data-match-media:is-dark__case.kebab="'prefers-color-scheme: dark'"></div>
```

### `data-on-raf` [Pro](https://data-star.dev/pro "Datastar Pro")

Runs an expression on every [`requestAnimationFrame`](https://developer.mozilla.org/en-US/docs/Web/API/Window/requestAnimationFrame) event.

```
<div data-on-raf="$count++"></div>
```

#### Modifiers

Modifiers allow you to modify the timing of the event listener.

- `__throttle` – Throttle the event listener.
  
  - `.500ms` – Throttle for 500 milliseconds (accepts any integer).
  - `.1s` – Throttle for 1 second (accepts any integer).
  - `.noleading` – Throttle without leading edge (must come after timing).
  - `.trailing` – Throttle with trailing edge (must come after timing).

```
<div data-on-raf__throttle.10ms="$count++"></div>
```

### `data-on-resize` [Pro](https://data-star.dev/pro "Datastar Pro")

Runs an expression whenever an element’s dimensions change.

```
<div data-on-resize="$count++"></div>
```

#### Modifiers

Modifiers allow you to modify the timing of the event listener.

- `__debounce` – Debounce the event listener.
  
  - `.500ms` – Debounce for 500 milliseconds (accepts any integer).
  - `.1s` – Debounce for 1 second (accepts any integer).
  - `.leading` – Debounce with leading edge (must come after timing).
  - `.notrailing` – Debounce without trailing edge (must come after timing).
- `__throttle` – Throttle the event listener.
  
  - `.500ms` – Throttle for 500 milliseconds (accepts any integer).
  - `.1s` – Throttle for 1 second (accepts any integer).
  - `.noleading` – Throttle without leading edge (must come after timing).
  - `.trailing` – Throttle with trailing edge (must come after timing).

```
<div data-on-resize__debounce.10ms="$count++"></div>
```

### `data-persist` [Pro](https://data-star.dev/pro "Datastar Pro")

Persists signals in [local storage](https://developer.mozilla.org/en-US/docs/Web/API/Window/localStorage). This is useful for storing values between page loads.

```
<div data-persist></div>
```

The signals to be persisted can be filtered by providing a value that is an object with `include` and/or `exclude` properties that are regular expressions.

```
<div data-persist="{include: /foo/, exclude: /bar/}"></div>
```

You can use a custom storage key by adding it after `data-persist:`. By default, signals are stored using the key `datastar`.

```
<div data-persist:mykey></div>
```

#### Modifiers

Modifiers allow you to modify the storage target.

- `__session` – Persists signals in [session storage](https://developer.mozilla.org/en-US/docs/Web/API/Window/sessionStorage) instead of local storage.

```
<!-- Persists signals using a custom key `mykey` in session storage -->
<div data-persist:mykey__session></div>
```

### `data-query-string` [Pro](https://data-star.dev/pro "Datastar Pro")

Syncs query string params to signal values on page load, and syncs signal values to query string params on change.

```
<div data-query-string></div>
```

The signals to be synced can be filtered by providing a value that is an object with `include` and/or `exclude` properties that are regular expressions.

```
<div data-query-string="{include: /foo/, exclude: /bar/}"></div>
```

#### Modifiers

Modifiers allow you to enable history support.

- `__filter` – Filters out empty values when syncing signal values to query string params.
- `__history` – Enables history support – each time a matching signal changes, a new entry is added to the browser’s history stack. Signal values are restored from the query string params on popstate events.

```
<div data-query-string__filter__history></div>
```

### `data-replace-url` [Pro](https://data-star.dev/pro "Datastar Pro")

Replaces the URL in the browser without reloading the page. The value can be a relative or absolute URL, and is an evaluated expression.

```
<div data-replace-url="`/page${page}`"></div>
```

### `data-scroll-into-view` [Pro](https://data-star.dev/pro "Datastar Pro")

Scrolls the element into view. Useful when updating the DOM from the backend, and you want to scroll to the new content.

```
<div data-scroll-into-view></div>
```

#### Modifiers

Modifiers allow you to modify scrolling behavior.

- `__smooth` – Scrolling is animated smoothly.
- `__instant` – Scrolling is instant.
- `__auto` – Scrolling is determined by the computed `scroll-behavior` CSS property.
- `__hstart` – Scrolls to the left of the element.
- `__hcenter` – Scrolls to the horizontal center of the element.
- `__hend` – Scrolls to the right of the element.
- `__hnearest` – Scrolls to the nearest horizontal edge of the element.
- `__vstart` – Scrolls to the top of the element.
- `__vcenter` – Scrolls to the vertical center of the element.
- `__vend` – Scrolls to the bottom of the element.
- `__vnearest` – Scrolls to the nearest vertical edge of the element.
- `__focus` – Focuses the element after scrolling.

### `data-view-transition` [Pro](https://data-star.dev/pro "Datastar Pro")

Sets the `view-transition-name` style attribute explicitly.

```
<div data-view-transition="$foo"></div>
```

Page level transitions are automatically handled by an injected meta tag. Inter-page elements are automatically transitioned if the [View Transition API](https://developer.mozilla.org/en-US/docs/Web/API/View_Transitions_API) is available in the browser and `useViewTransitions` is `true`.

## Attribute Evaluation Order 

Elements are evaluated by walking the DOM in a depth-first manner, and attributes are applied in the order they appear in the element. This is important in some cases, such as when using `data-indicator` with a fetch request initiated in a `data-init` attribute, in which the indicator signal must be created before the fetch request is initialized.

```
<div data-indicator:fetching data-init="@get('/endpoint')"></div>
```

Data attributes are evaluated and applied on page load (after Datastar has initialized), and are reapplied after any DOM patches that add, remove, or change them. Note that [morphing elements](https://data-star.dev/reference/sse_events#datastar-patch-elements) preserves existing attributes unless they are explicitly changed in the DOM, meaning they will only be reapplied if the attribute itself is changed.

## Attribute Casing 

[According to the HTML spec](https://developer.mozilla.org/en-US/docs/Web/HTML/Global_attributes/data-*), all `data-*` attributes (not Datastar the framework, but any time a data attribute appears in the DOM) are case-insensitive. When Datastar processes these attributes, hyphenated names are automatically converted to [camel case](https://developer.mozilla.org/en-US/docs/Glossary/Camel_case) by removing hyphens and uppercasing the letter following each hyphen.

Datastar handles casing of data attribute key suffixes containing hyphens in two ways:
. The keys used in attributes that define signals (`data-bind:*`, `data-signals:*`, `data-computed:*`, etc.), are converted to camel case (the recommended casing for signals) by removing hyphens and uppercasing the letter following each hyphen. For example, `data-signals:my-signal` defines a signal named `mySignal`, and you would use the signal in a [Datastar expression](https://data-star.dev/guide/datastar_expressions) as `$mySignal`.
. The keys suffixes used by all other attributes are, by default, converted to [kebab case](https://developer.mozilla.org/en-US/docs/Glossary/Kebab_case). For example, `data-class:text-blue-700` adds or removes the class `text-blue-700`, and `data-on:rocket-launched` would react to the event named `rocket-launched`.

You can use the `__case` modifier to convert between `camelCase`, `kebab-case`, `snake_case`, and `PascalCase`, or alternatively use object syntax when available.

For example, if listening for an event called `widgetLoaded`, you would use `data-on:widget-loaded__case.camel`.

## Aliasing Attributes 

It is possible to alias `data-*` attributes to a custom alias (`data-alias-*`, for example) using the [bundler](https://data-star.dev/pro/bundler). A custom alias should *only* be used if you have a conflict with a legacy library and [`data-ignore`](#data-ignore) cannot be used.

We maintain a `data-star-*` aliased version that can be included as follows.

```
<script type="module" src="https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.1/bundles/datastar-aliased.js"></script>
```

## Datastar Expressions 

Datastar expressions used in `data-*` attributes parse signals, converting all dollar signs followed by valid signal name characters into their corresponding signal values. Expressions support standard JavaScript syntax, including operators, function calls, ternary expressions, and object and array literals.

A variable `el` is available in every Datastar expression, representing the element that the attribute exists on.

```
<div id="bar" data-text="$foo + el.id"></div>
```

Read more about [Datastar expressions](https://data-star.dev/guide/datastar_expressions) in the guide.

## Error Handling 

Datastar has built-in error handling and reporting for runtime errors. When a data attribute is used incorrectly, for example `data-text-foo`, the following error message is logged to the browser console.

```
Uncaught datastar runtime error: textKeyNotAllowed
More info: https://data-star.dev/errors/key_not_allowed?metadata=%7B%22plugin%22%3A%7B%22name%22%3A%22text%22%2C%22type%22%3A%22attribute%22%7D%2C%22element%22%3A%7B%22id%22%3A%22%22%2C%22tag%22%3A%22DIV%22%7D%2C%22expression%22%3A%7B%22rawKey%22%3A%22textFoo%22%2C%22key%22%3A%22foo%22%2C%22value%22%3A%22%22%2C%22fnContent%22%3A%22%22%7D%7D
Context: {
    "plugin": {
        "name": "text",
        "type": "attribute"
    },
    "element": {
        "id": "",
        "tag": "DIV"
    },
    "expression": {
        "rawKey": "textFoo",
        "key": "foo",
        "value": "",
        "fnContent": ""
    }
}
```

The “More info” link takes you directly to a context-aware error page that explains the error and provides correct sample usage. See [the error page for the example above](https://data-star.dev/errors/key_not_allowed?metadata=%7B%22plugin%22%3A%7B%22name%22%3A%22text%22%2C%22type%22%3A%22attribute%22%7D%2C%22element%22%3A%7B%22id%22%3A%22%22%2C%22tag%22%3A%22DIV%22%7D%2C%22expression%22%3A%7B%22rawKey%22%3A%22textFoo%22%2C%22key%22%3A%22foo%22%2C%22value%22%3A%22%22%2C%22fnContent%22%3A%22%22%7D%7D), and all available error messages in the sidebar menu.

# Actions

Datastar provides actions (helper functions) that can be used in Datastar expressions.

> The `@` prefix designates actions that are safe to use in expressions. This is a security feature that prevents arbitrary JavaScript from being executed in the browser. Datastar uses [`Function()` constructors](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Function/Function) to create and execute these actions in a secure and controlled sandboxed environment.

### `@peek()` 

> `@peek(callable: () => any)`

Allows accessing signals without subscribing to their changes in expressions.

```
<div data-text="$foo + @peek(() => $bar)"></div>
```

In the example above, the expression in the `data-text` attribute will be re-evaluated whenever `$foo` changes, but it will *not* be re-evaluated when `$bar` changes, since it is evaluated inside the `@peek()` action.

### `@setAll()` 

> `@setAll(value: any, filter?: {include: RegExp, exclude?: RegExp})`

Sets the value of all matching signals (or all signals if no filter is used) to the expression provided in the first argument. The second argument is an optional filter object with an `include` property that accepts a regular expression to match signal paths. You can optionally provide an `exclude` property to exclude specific patterns.

> The [Datastar Inspector](https://data-star.dev/pro#datastar-inspector) can be used to inspect and filter current signals and view signal patch events in real-time.

```
<!-- Sets the `foo` signal only -->
<div data-signals:foo="false">
    <button data-on:click="@setAll(true, {include: /^foo$/})"></button>
</div>

<!-- Sets all signals starting with `user.` -->
<div data-signals="{user: {name: '', nickname: ''}}">
    <button data-on:click="@setAll('johnny', {include: /^user\./})"></button>
</div>

<!-- Sets all signals except those ending with `_temp` -->
<div data-signals="{data: '', data_temp: '', info: '', info_temp: ''}">
    <button data-on:click="@setAll('reset', {include: /.*/, exclude: /_temp$/})"></button>
</div>
```

### `@toggleAll()` 

> `@toggleAll(filter?: {include: RegExp, exclude?: RegExp})`

Toggles the boolean value of all matching signals (or all signals if no filter is used). The argument is an optional filter object with an `include` property that accepts a regular expression to match signal paths. You can optionally provide an `exclude` property to exclude specific patterns.

> The [Datastar Inspector](https://data-star.dev/pro#datastar-inspector) can be used to inspect and filter current signals and view signal patch events in real-time.

```
<!-- Toggles the `foo` signal only -->
<div data-signals:foo="false">
    <button data-on:click="@toggleAll({include: /^foo$/})"></button>
</div>

<!-- Toggles all signals starting with `is` -->
<div data-signals="{isOpen: false, isActive: true, isEnabled: false}">
    <button data-on:click="@toggleAll({include: /^is/})"></button>
</div>

<!-- Toggles signals starting with `settings.` -->
<div data-signals="{settings: {darkMode: false, autoSave: true}}">
    <button data-on:click="@toggleAll({include: /^settings\./})"></button>
</div>
```

## Backend Actions 

### `@get()` 

> `@get(uri: string, options={ })`

Sends a `GET` request to the backend using the [Fetch API](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API). The URI can be any valid endpoint and the response must contain zero or more [Datastar SSE events](https://data-star.dev/reference/sse_events).

```
<button data-on:click="@get('/endpoint')"></button>
```

By default, requests are sent with a `Datastar-Request: true` header, and a `{datastar: *}` object containing all existing signals, except those beginning with an underscore. This behavior can be changed using the [`filterSignals`](#filterSignals) option, which allows you to include or exclude specific signals using regular expressions.

> When using a `get` request, the signals are sent as a query parameter, otherwise they are sent as a JSON body.

When a page is hidden (in a background tab, for example), the default behavior for `get` requests is for the SSE connection to be closed, and reopened when the page becomes visible again. To keep the connection open when the page is hidden, set the [`openWhenHidden`](#openWhenHidden) option to `true`.

```
<button data-on:click="@get('/endpoint', {openWhenHidden: true})"></button>
```

It’s possible to send form encoded requests by setting the `contentType` option to `form`. This sends requests using `application/x-www-form-urlencoded` encoding.

```
<button data-on:click="@get('/endpoint', {contentType: 'form'})"></button>
```

It’s also possible to send requests using `multipart/form-data` encoding by specifying it in the `form` element’s [`enctype`](https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement/enctype) attribute. This should be used when uploading files. See the [form data example](https://data-star.dev/examples/form_data).

```
<form enctype="multipart/form-data">
    <input type="file" name="file" />
    <button data-on:click="@post('/endpoint', {contentType: 'form'})"></button>
</form>
```

### `@post()` 

> `@post(uri: string, options={ })`

Works the same as [`@get()`](#get) but sends a `POST` request to the backend.

```
<button data-on:click="@post('/endpoint')"></button>
```

### `@put()` 

> `@put(uri: string, options={ })`

Works the same as [`@get()`](#get) but sends a `PUT` request to the backend.

```
<button data-on:click="@put('/endpoint')"></button>
```

### `@patch()` 

> `@patch(uri: string, options={ })`

Works the same as [`@get()`](#get) but sends a `PATCH` request to the backend.

```
<button data-on:click="@patch('/endpoint')"></button>
```

### `@delete()` 

> `@delete(uri: string, options={ })`

Works the same as [`@get()`](#get) but sends a `DELETE` request to the backend.

```
<button data-on:click="@delete('/endpoint')"></button>
```

### Options 

All of the actions above take a second argument of options.

- `contentType` – The type of content to send. A value of `json` sends all signals in a JSON request. A value of `form` tells the action to look for the closest form to the element on which it is placed (unless a `selector` option is provided), perform validation on the form elements, and send them to the backend using a form request (no signals are sent). Defaults to `json`.
- `filterSignals` – A filter object with an `include` property that accepts a regular expression to match signal paths (defaults to all signals: `/.*/`), and an optional `exclude` property to exclude specific signal paths (defaults to all signals that do not have a `_` prefix: `/(^_|\._).*/`).
  
  > The [Datastar Inspector](https://data-star.dev/pro#datastar-inspector) can be used to inspect and filter current signals and view signal patch events in real-time.
- `selector` – Optionally specifies a form to send when the `contentType` option is set to `form`. If the value is `null`, the closest form is used. Defaults to `null`.
- `headers` – An object containing headers to send with the request.
- `openWhenHidden` – Whether to keep the connection open when the page is hidden. Useful for dashboards but can cause a drain on battery life and other resources when enabled. Defaults to `false` for `get` requests, and `true` for all other HTTP methods.
- `payload` – Allows the fetch payload to be overridden with a custom object.
- `retry` – Determines when to retry requests. Can be `'auto'` (default, retries on network errors only), `'error'` (retries on `4xx` and `5xx` responses), `'always'` (retries on all non-`204` responses except redirects), or `'never'` (disables retries). Defaults to `'auto'`.
- `retryInterval` – The retry interval in milliseconds. Defaults to `1000` (one second).
- `retryScaler` – A numeric multiplier applied to scale retry wait times. Defaults to `2`.
- `retryMaxWait` – The maximum allowable wait time in milliseconds between retries. Defaults to `30000` (30 seconds).
- `retryMaxCount` – The maximum number of retry attempts. Defaults to `10`.
- `requestCancellation` – Controls request cancellation behavior. Can be `'auto'` (default, cancels existing requests on the same element), `'cleanup'` (cancels existing requests on the same element and on element or attribute cleanup), `'disabled'` (allows concurrent requests), or an `AbortController` instance for custom control. Defaults to `'auto'`.

```
<button data-on:click="@get('/endpoint', {
    filterSignals: {include: /^foo\./},
    headers: {
        'X-Csrf-Token': 'JImikTbsoCYQ9oGOcvugov0Awc5LbqFsZW6ObRCxuq',
    },
    openWhenHidden: true,
    requestCancellation: 'disabled',
})"></button>
```

### Request Cancellation 

By default, when a new fetch request is initiated on an element, any existing request on that same element is automatically cancelled. This prevents multiple concurrent requests from conflicting with each other and ensures clean state management.

For example, if a user rapidly clicks a button that triggers a backend action, only the most recent request will be processed:

```
<!-- Clicking this button multiple times will cancel previous requests (default behavior) -->
<button data-on:click="@get('/slow-endpoint')">Load Data</button>
```

This automatic cancellation happens at the element level, meaning requests on different elements can run concurrently without interfering with each other.

You can control this behavior using the [`requestCancellation`](#requestCancellation) option:

```
<!-- Allow concurrent requests (no automatic cancellation) -->
<button data-on:click="@get('/endpoint', {requestCancellation: 'disabled'})">Allow Multiple</button>

<!-- Custom abort controller for fine-grained control -->
<div data-signals:controller="new AbortController()">
    <button data-on:click="@get('/endpoint', {requestCancellation: $controller})">Start Request</button>
    <button data-on:click="$controller.abort()">Cancel Request</button>
</div>
```

### Response Handling 

Backend actions automatically handle different response content types:

- `text/event-stream` – Standard SSE responses with [Datastar SSE events](https://data-star.dev/reference/sse_events).
- `text/html` – HTML elements to patch into the DOM.
- `application/json` – JSON encoded signals to patch.
- `text/javascript` – JavaScript code to execute in the browser.

#### `text/html`

When returning HTML (`text/html`), the server can optionally include the following response headers:

- `datastar-selector` – A CSS selector for the target elements to patch
- `datastar-mode` – How to patch the elements (`outer`, `inner`, `remove`, `replace`, `prepend`, `append`, `before`, `after`). Defaults to `outer`.
- `datastar-use-view-transition` – Whether to use the [View Transition API](https://developer.mozilla.org/en-US/docs/Web/API/View_Transitions_API) when patching elements.

```
response.headers.set('Content-Type', 'text/html')
response.headers.set('datastar-selector', '#my-element')
response.headers.set('datastar-mode', 'inner')
response.body = '<p>New content</p>'
```

#### `application/json`

When returning JSON (`application/json`), the server can optionally include the following response header:

- `datastar-only-if-missing` – If set to `true`, only patch signals that don’t already exist.

```
response.headers.set('Content-Type', 'application/json')
response.headers.set('datastar-only-if-missing', 'true')
response.body = JSON.stringify({ foo: 'bar' })
```

#### `text/javascript`

When returning JavaScript (`text/javascript`), the server can optionally include the following response header:

- `datastar-script-attributes` – Sets the script element’s attributes using a JSON encoded string.

```
response.headers.set('Content-Type', 'text/javascript')
response.headers.set('datastar-script-attributes', JSON.stringify({ type: 'module' }))
response.body = 'console.log("Hello from server!");'
```

### Events 

All of the actions above trigger `datastar-fetch` events during the fetch request lifecycle. The event type determines the stage of the request.

- `started` – Triggered when the fetch request is started.
- `finished` – Triggered when the fetch request is finished.
- `error` – Triggered when the fetch request encounters an error.
- `retrying` – Triggered when the fetch request is retrying.
- `retries-failed` – Triggered when all fetch retries have failed.

```
<div data-on:datastar-fetch="
    evt.detail.type === 'error' && console.log('Fetch error encountered')
"></div>
```

## Pro Actions 

### `@clipboard()` [Pro](https://data-star.dev/pro "Datastar Pro")

> `@clipboard(text: string, isBase64?: boolean)`

Copies the provided text to the clipboard. If the second parameter is `true`, the text is treated as [Base64](https://developer.mozilla.org/en-US/docs/Glossary/Base64) encoded, and is decoded before copying.

> Base64 encoding is useful when copying content that contains special characters, quotes, or code fragments that might not be valid within HTML attributes. This prevents parsing errors and ensures the content is safely embedded in `data-*` attributes.

```
<!-- Copy plain text -->
<button data-on:click="@clipboard('Hello, world!')"></button>

<!-- Copy base64 encoded text (will decode before copying) -->
<button data-on:click="@clipboard('SGVsbG8sIHdvcmxkIQ==', true)"></button>
```

### `@fit()` [Pro](https://data-star.dev/pro "Datastar Pro")

> `@fit(v: number, oldMin: number, oldMax: number, newMin: number, newMax: number, shouldClamp=false, shouldRound=false)`

Linearly interpolates a value from one range to another. This is useful for converting between different scales, such as mapping a slider value to a percentage or converting temperature units.

The optional `shouldClamp` parameter ensures the result stays within the new range, and `shouldRound` rounds the result to the nearest integer.

```
<!-- Convert a 0-100 slider to 0-255 RGB value -->
<div>
    <input type="range" min="0" max="100" value="50" data-bind:slider-value>
    <div data-computed:rgb-value="@fit($sliderValue, 0, 100, 0, 255)">
        RGB Value: <span data-text="$rgbValue"></span>
    </div>
</div>

<!-- Convert Celsius to Fahrenheit -->
<div>
    <input type="number" data-bind:celsius value="20" />
    <div data-computed:fahrenheit="@fit($celsius, 0, 100, 32, 212)">
        <span data-text="$celsius"></span>°C = <span data-text="$fahrenheit.toFixed(1)"></span>°F
    </div>
</div>

<!-- Map mouse position to element opacity (clamped) -->
<div
    data-signals:mouse-x="0"
    data-computed:opacity="@fit($mouseX, 0, window.innerWidth, 0, 1, true)"
    data-on:mousemove__window="$mouseX = evt.clientX"
    data-attr:style="'opacity: ' + $opacity"
>
    Move your mouse horizontally to change opacity
</div>
```

### `@intl()` [Pro](https://data-star.dev/pro "Datastar Pro")

> `@intl(type: string, value: any, options?: Record<string, any>, locale?: string | string[])`

Provides internationalized, locale-aware formatting for dates, numbers, and other values using the [Intl namespace object](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl).

```
<!-- Converts a number to a formatted USD currency string in the user’s locale -->
<div data-text="@intl('number', 1000000, {style: 'currency', currency: 'USD'})"></div>

<!-- Converts a date to a formatted string in the specified locale -->
<div data-text="@intl('datetime', new Date(), {weekday: 'long', year: 'numeric', month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit'}, 'de-AT')"></div>
```

The `type` parameter specifies the type of value to format. Possible values:

- `datetime` – formats dates and times using [Intl.DateTimeFormat](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat)
- `number` – formats numbers using [Intl.NumberFormat](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/NumberFormat)
- `pluralRules` – determines plural rules using [Intl.PluralRules](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/PluralRules)
- `relativeTime` – formats relative time using [Intl.RelativeTimeFormat](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/RelativeTimeFormat)
- `list` – formats lists using [Intl.ListFormat](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/ListFormat)
- `displayNames` – gets display names for languages, regions, etc. using [Intl.DisplayNames](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DisplayNames)

The `options` parameter can be one of several [Intl option types](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl#options_argument) depending on the type parameter:

- [Intl.DateTimeFormatOptions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/DateTimeFormat#options) – when type is `datetime`
- [Intl.NumberFormatOptions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/NumberFormat/NumberFormat#options) – when type is `number`
- [Intl.PluralRulesOptions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/PluralRules/PluralRules#options) – when type is `pluralRules`
- [Intl.RelativeTimeFormatOptions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/RelativeTimeFormat/RelativeTimeFormat#options) – when type is `relativeTime`
- [Intl.ListFormatOptions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/ListFormat/ListFormat#options) – when type is `list`
- [Intl.DisplayNamesOptions](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DisplayNames/DisplayNames#options) – when type is `displayNames`

# Rocket

Rocket is currently in beta.

## Overview 

Rocket is Datastar Pro’s web-component API. You define a custom element with `rocket(tagName, { ... })`, describe public props with codecs, put non-DOM instance behavior in `setup`, use `onFirstRender` only when work depends on rendered refs or mounted DOM, and return DOM from `render`.

Rocket is a JavaScript API built around the browser’s custom-element model, with Datastar handling reactivity, local signal scoping, action dispatch, and DOM application.

```
rocket('demo-counter', {
  mode: 'light',
  props: ({ number, string }) => ({
    count: number.step(1).min(0),
    label: string.trim.default('Counter'),
  }),
  setup: ({ $$, observeProps, props }) => {
    $$.count = props.count
    observeProps(() => {
      $$.count = props.count
    }, 'count')
  },
  render: ({ html, props: { count, label } }) => {
    return html`
      <div class="stack gap-2">
        <button
          type="button"
          data-on:click="$$count += 1"
          data-text="${label} + ': ' + $$count"
        ></button>
        <template data-if="$$count !== ${count}">
          <button type="button" data-on:click="$$count = ${count}">Reset</button>
        </template>
      </div>
    `
  },
})
```

The result is a real web component. Once defined, you use it like any other custom element.

```
<script type="module">
  import { rocket } from '/bundles/datastar-pro.js'
  // define demo-counter here
</script>

<demo-counter count="5" label="Inventory"></demo-counter>
```

The examples on this page assume the same module pattern. Import `rocket` explicitly from the bundle module rather than relying on a global.

### `tag` 

`tag` is the first argument to `rocket(...)`. It must contain a hyphen, must be unique, and becomes the actual HTML tag users place in the page.

```
rocket(tag: string, options?: RocketDefinition<Defs>): void
```

```
rocket('demo-user-card', {
  props: ({ string }) => ({
    name: string.default('Anonymous'),
  }),
  render: ({ html, props: { name } }) => {
    return html`
        <p>${name}</p>
    `
  },
})
```

Rocket registers the element with `customElements.define`. Re-registering the same tag is ignored, which makes repeated module evaluation safe during development.

## Definition 

A Rocket definition describes one custom-element type. Each field controls a specific part of the browser component lifecycle.

FieldWhat it does `mode`Chooses light DOM, open shadow DOM, or closed shadow DOM.`props`Defines public props, decoding, defaults, and attribute reflection.`setup`Runs once per connected instance to create local state, prop observers, timers, host APIs, and cleanup.`onFirstRender`Runs after the initial render and Datastar apply pass, when refs and rendered DOM exist.`render`Returns the component DOM as Rocket `html` or `svg`.`renderOnPropChange`Controls whether prop updates trigger rerendering.`manifest`Adds slot and event metadata to Rocket’s generated component manifest.

### `mode` 

`mode` chooses where the component renders.

```
mode?: 'open' | 'closed' | 'light'
```

ValueMount targetWhen to use it `'light'`The host element itself.Use when the component should participate directly in the page’s DOM and CSS.`'open'`An open shadow root.Use when you want style encapsulation but still want `element.shadowRoot`.`'closed'`A closed shadow root.Use when the internal DOM should stay fully encapsulated.

If you omit `mode`, Rocket uses shadow DOM and defaults to `'open'`.

Rocket defaults to `'open'` because it gives you native slots and a normal shadow-root debugging surface without giving up the main styling mechanism components should rely on: CSS custom properties. CSS variables pierce the shadow boundary, so the usual pattern is to keep component structure encapsulated while letting parents theme it through `--tokens`.

### `props` 

`props` defines the component’s public API. Rocket calls it once at definition time and passes in the codec registry. The object you return becomes:

```
props?: (codecs: CodecRegistry) => Defs
```

- The list of observed HTML attributes.
- The normalization pipeline for incoming attribute values.
- The property descriptors on the custom-element instance.
- The typed `props` object passed to `setup` and `render`.

```
rocket('demo-progress', {
  props: ({ number, bool, string }) => ({
    value: number.clamp(0, 100),
    striped: bool,
    label: string.trim.default('Progress'),
  }),
  render: ({ html, props: { value, striped, label } }) => {
    return html`
      <div>
        <strong>${label}</strong>
        <div class="bar" data-style="{ width: ${value} + '%' }"></div>
        <span data-show="${striped}">Striped</span>
      </div>
    `
  },
})
```

`props` is optional. If you omit it, Rocket defines no observed attributes and `setup` / `render` receive an empty decoded props object.

Each declared prop already gets a normal element property accessor on the custom-element prototype. In other words, `host.value`, `host.checked`, and similar prop access work by default without per-instance `Object.defineProperty(...)` calls.

The intent of `props` is to give Rocket components a decoded props object that is always usable. Rocket codecs normalize incoming attribute values into valid decoded values instead of throwing on bad input.

On the DOM side, web-component attributes arrive as strings. Rocket converts those raw attribute strings into usable prop types for you before `setup` or `render` reads them.

This follows the same broad design instinct as Go and Odin zero values and the general behavior of native HTML elements: invalid or missing input should degrade to a sensible value, not crash component setup or rendering. A malformed number becomes a number, a missing string becomes a string, and defaults stay inside the codec layer instead of being rechecked throughout the component.

Prop names are written in JavaScript style and reflected to attributes in kebab case. A prop named `startDate` maps to an HTML attribute named `start-date`.

### Props vs Signals 

Default to `props` when data is part of the component’s public API. Props already match the native browser model: they map cleanly to attributes and element properties, reflect through the host element, and give outside code a normal way to configure the component.

Use local Rocket signals for internal reactive state and for imperative integration points where the component has to talk to something outside Rocket’s normal prop/render flow. Good examples are calling charting libraries, talking to third-party widgets, starting timers, or reacting to fetch results and other external async work.

A useful rule is: if a parent or page author should be able to set it directly on the element, make it a prop. If the value mainly exists so `setup` can coordinate external calls or internal UI state over time, make it a signal.

### `manifest` 

`manifest` lets you add documentation metadata that Rocket cannot infer from the DOM alone. Rocket already generates prop metadata from your codecs. Use `manifest` to document slots and events so tooling can describe the full public surface of the component.

```
manifest?: {
  slots?: Array<{
    name: string
    description?: string
  }>
  events?: Array<{
    name: string
    kind?: 'event' | 'custom-event'
    bubbles?: boolean
    composed?: boolean
    description?: string
  }>
}
```

```
import { publishRocketManifests, rocket } from '/bundles/datastar-pro.js'

rocket('demo-dialog', {
  props: ({ string, bool }) => ({
    title: string.default('Dialog'),
    open: bool,
  }),
  manifest: {
    slots: [
      { name: 'default', description: 'Dialog body content.' },
      { name: 'footer', description: 'Action row content.' },
    ],
    events: [
      {
        name: 'close',
        kind: 'custom-event',
        bubbles: true,
        composed: true,
        description: 'Fired when the dialog requests dismissal.',
      },
    ],
  },
  render: ({ html, props: { title, open } }) => {
    return html`
      <section data-show="${open}">
        <header>${title}</header>
        <slot></slot>
        <footer><slot name="footer"></slot></footer>
      </section>
    `
  },
})

const manifest = customElements.get('demo-dialog')?.manifest?.()

await publishRocketManifests({
  endpoint: '/api/rocket/manifests',
})
```

The generated manifest includes one component entry per Rocket tag. Each entry contains the tag name, inferred prop metadata, and your manual slot and event metadata.

Each registered Rocket class also gets a static `manifest()` method that returns that component’s manifest entry. This is useful when you want to inspect or test one component locally without publishing the full document.

`publishRocketManifests(...)` posts the full manifest document as JSON. Rocket sorts components by tag and includes a top-level `version` and `generatedAt` timestamp so a docs build or registry service can store snapshots.

### `setup` 

`setup` runs once per connected element instance after Rocket creates the component scope and before the initial Datastar apply pass. This is the default place for local signals, computed state, prop observers, timers, cleanup handlers, and host APIs that do not depend on rendered refs.

If the code needs rendered DOM, measurements, focus targets, or `data-ref:*` handles, move that part to `onFirstRender()` instead of delaying it inside `setup()`.

```
setup?: (context: SetupContext<InferProps<Defs>>) => void
```

Datastar is mostly about setting up relationships, not manually pushing DOM updates. Most component behavior should come from local signals changing over time and the rest of the component reacting to those changes through effects, bindings, and render output.

```
rocket('demo-timer', {
  props: ({ number, bool }) => ({
    intervalMs: number.min(50).default(1000),
    autoplay: bool,
  }),
  setup: ({ $$, cleanup, props, observeProps }) => {
    $$.seconds = 0
    let timerId = 0

    let syncTimer = () => {
      clearInterval(timerId)
      if (!props.autoplay) {
        return
      }
      timerId = window.setInterval(() => {
        $$.seconds += 1
      }, props.intervalMs)
    }

    syncTimer()
    observeProps(syncTimer)

    cleanup(() => clearInterval(timerId))
  },
  render: ({ html }) => {
    return html`
        <p data-text="$$seconds"></p>
    `
  },
})
```

Use `setup` to handle non-ref behavior. Keep markup creation in `render`, and keep ref-backed DOM work in `onFirstRender()`. That split matters because Rocket may rerun `render` many times, while `setup` and `onFirstRender` only run once per connected instance.

When behavior needs to react to prop changes, use `observeProps(...)`. The `props` object is normalized and always usable, but it is not itself a local signal source.

### `onFirstRender` 

`onFirstRender` runs once per connected instance after Rocket has finished the initial `render()`, Datastar `apply(...)`, and ref population pass. Use it for work that depends on rendered DOM or `data-ref:*` refs.

```
onFirstRender?: (context: SetupContext<InferProps<Defs>> & { refs: Record<string, any> }) => void
```

This is the right place for ref-backed host accessors, DOM measurements, focus management, or third-party widget setup that needs the actual rendered nodes. If a piece of logic would work without refs, keep it in `setup` instead.

`onFirstRender` receives the normal setup context plus `refs`, so it can use `$$`, `refs`, `overrideProp`, `defineHostProp`, `cleanup`, and the rest without nesting a second callback inside setup.

```
rocket('demo-input-bridge', {
  props: ({ string }) => ({
    value: string.default(''),
  }),
  render: ({ html, props: { value } }) => html`
    <input data-ref:input value="${value}">
  `,
  onFirstRender: ({ overrideProp, refs }) => {
    overrideProp(
      'value',
      (getDefault) => refs.input?.value ?? getDefault(),
      (value, setDefault) => {
        const next = String(value ?? '')
        if (refs.input && refs.input.value !== next) refs.input.value = next
        setDefault(next)
      },
    )
  },
})
```

If setup code needs to force a render later, call `ctx.render` with an empty overrides object and any trailing args. That reruns the component render function with the current host, props, and template helpers, plus any extra arguments you pass.

This is not a replacement for local signals. Reach for `ctx.render` when async or imperative work needs a coarse structural patch, similar to switching a `data-if` branch. For high-frequency state like counters, form values, loading flags, or selection state, keep using signals and normal Datastar bindings.

```
rocket('demo-user-card', {
  props: ({ string, number }) => ({
    userId: number.min(1),
    fallbackName: string.default('Unknown user'),
  }),
  setup: ({ cleanup, render, props }) => {
    let cancelled = false

    ;(async () => {
      try {
        const response = await fetch('/users/' + props.userId + '.json')
        const user = await response.json()
        if (!cancelled) {
          render({}, user, null)
        }
      } catch (error) {
        if (!cancelled) {
          render({}, null, error)
        }
      }
    })()

    cleanup(() => {
      cancelled = true
    })
  },
  render: ({ html, props: { fallbackName } }, user = null, error = null) => {
    if (error) {
      return html`<p>Failed to load user.</p>`
    }
    if (!user) {
      return html`<p>Loading user...</p>`
    }
    return html`
      <article>
        <h3>${user.name ?? fallbackName}</h3>
        <p>${user.email ?? 'No email provided'}</p>
      </article>
    `
  },
})
```

### `render` 

`render` is optional. It receives the normalized props, the host element, and two tagged-template helpers: `html` and `svg`. Return a Rocket tagged-template fragment, primitive text, an iterable of composed values, or `null`/`undefined`.

```
type RocketPrimitiveRenderValue =
  | string
  | number
  | boolean
  | bigint
  | Date
  | null
  | undefined

type RocketComposedRenderValue =
  | RocketPrimitiveRenderValue
  | Node
  | Iterable<RocketComposedRenderValue>

type RocketRenderValue =
  | DocumentFragment
  | RocketPrimitiveRenderValue
  | Iterable<RocketComposedRenderValue>

type RocketRender<Props extends Record<string, any>> = {
  (context: RenderContext<Props>): RocketRenderValue
  <A1>(context: RenderContext<Props>, a1: A1): RocketRenderValue
  <A1, A2, A3, A4, A5, A6, A7, A8>(
    context: RenderContext<Props>,
    a1: A1,
    a2: A2,
    a3: A3,
    a4: A4,
    a5: A5,
    a6: A6,
    a7: A7,
    a8: A8,
  ): RocketRenderValue
}

render?: RocketRender<InferProps<Defs>>
```

This is the method that turns Rocket from a state container into an actual web component. The host element stays stable, while the rendered subtree inside it or inside its shadow root is morphed from the output of `render`.

Rocket supports up to 8 typed trailing render arguments. Automatic renders call `render(context)`. Setup-driven renders can call `ctx.render` with an empty overrides object to pass those extra values explicitly.

Treat those manual render calls like coarse DOM branch updates, not like a second reactive state system. If the UI should keep updating as values change over time, model that state with signals and let Datastar update the existing DOM in place.

Inside Rocket `html` templates, attribute interpolation omits the attribute for `false`, `null`, and `undefined`, while `true` creates the empty-string form of a boolean attribute.

In normal data positions, `false`, `null`, and `undefined` render nothing. If you want the literal text `"false"` or `"true"` in the DOM, pass a string, not a boolean value.

If you omit `render`, Rocket still registers the custom element, runs `setup`, scopes host-owned children, and wires action dispatch, but it does not morph a rendered subtree.

### `renderOnPropChange` 

```
renderOnPropChange?:
  | boolean
  | ((context: {
      host: HTMLElement
      props: Props
      changes: Partial<Props>
    }) => boolean)
```

By default Rocket behaves as if this were `true`.

Rocket coalesces multiple prop updates in the same turn into a single queued `render()` call per component. Prop changes still update `props` synchronously and notify `observeProps()` listeners immediately; the queue only deduplicates the component DOM rerender step.

```
rocket('demo-chart', {
  props: ({ json, string }) => ({
    series: json.default(() => []),
    theme: string.default('light'),
  }),
  mode: 'light',
  renderOnPropChange: ({ changes }) => {
    return 'theme' in changes
  },
  setup: ({ host, observeProps, props }) => {
    observeProps(() => {
      drawChart(host, props.series, props.theme)
    }, 'series', 'theme')
  },
  render: ({ html }) => {
    return html`
        <canvas width="640" height="320"></canvas>
    `
  },
})
```

In that pattern, updating `series` still updates `props.series` and notifies `observeProps` listeners, but it skips DOM rerendering because the canvas is updated imperatively. Use `observeProps` for prop changes; use `effect` for local Rocket signals or other reactive Datastar state.

## Modes 

Rocket supports both light DOM and shadow DOM because real component systems need both.

- Use `light` when the component should inherit page styles, participate in layout naturally, and expose its internals to outside CSS.
- Use `open` when you want encapsulated styles but still need debugging access through `shadowRoot`.
- Use `closed` when the component is a sealed implementation detail.

In shadow DOM, `<slot>` is the platform slot API. In light DOM, it is only a Rocket placeholder for host-child projection. Rocket supports default and named `<slot>` markers in light DOM, and if a slot receives no matching host children, its fallback content is rendered instead. This is still a Rocket runtime feature, not browser slotting.

```
rocket('demo-chip', {
  mode: 'light',
  props: ({ string }) => ({
    label: string.default('Chip'),
  }),
  render: ({ html, props: { label } }) => {
    return html`
        <span class="chip">${label}</span>
    `
  },
})

rocket('demo-modal-frame', {
  mode: 'open',
  props: ({ string }) => ({
    title: string.default('Dialog'),
  }),
  render: ({ html, props: { title } }) => {
    return html`
      <style>
        :host { display: block; }
        .frame { border: 1px solid #d4d4d8; padding: 1rem; }
      </style>
      <div class="frame">${title}</div>
    `
  },
})
```

## Props and Codecs 

Custom-element attributes arrive as strings. Rocket codecs turn those strings into useful values, apply normalization, supply defaults, and encode property writes back to attributes. This is what makes a Rocket component feel like a real typed component API instead of a stringly-typed DOM wrapper.

### How Props Flow 

Each prop has one codec. Rocket uses it in three places:

- At construction time, to decode initial attributes into `props`.
- When an observed attribute changes, to decode the new string value.
- When code assigns to `element.someProp`, to encode that value back into an attribute.

```
rocket('demo-badge', {
  props: ({ string, bool, number }) => ({
    label: string.trim.default('New'),
    tone: string.lower.default('neutral'),
    visible: bool.default(true),
    priority: number.clamp(0, 5),
  }),
  render: ({ html, props: { label, tone, visible, priority } }) => {
    return html`
      <span
        data-show="${visible}"
        data-attr:data-tone="${tone}"
        data-text="${label + ' #' + priority}">
      </span>
    `
  },
})

const badge = document.querySelector('demo-badge')

// Property write:
badge.priority = 7

// Reflected attribute after encoding:
// <demo-badge priority="5"></demo-badge>

// Attribute write:
badge.setAttribute('label', '  shipped  ')

// Decoded prop value in setup/render:
// props.label === 'shipped'
```

Defaults matter for component design because custom elements are often dropped into a page with incomplete markup. A default lets the component boot into a valid state without requiring every consumer to pass every attribute.

### Fluent Codec Pattern 

Codecs are immutable builders. Every method returns a new codec with an extra transform or constraint layered on top of the previous one. That is why the API is fluent.

```
props: ({ string, number, object, array, oneOf }) => ({
  slug: string.trim.lower.kebab.maxLength(48),
  progress: number.clamp(0, 100).step(5),
  theme: oneOf('light', 'dark', 'system').default('system'),
  tags: array(string.trim.lower),
  profile: object({
    name: string.trim.default('Anonymous'),
    age: number.min(0),
  }),
})
```

Read each chain left to right.

- `string.trim.lower.kebab` means “decode as string, trim it, lowercase it, then convert it to kebab case.”
- `number.clamp(0, 100).step(5)` means “decode as number, constrain it to 0-100, then snap it to increments of 5.”
- `array(string.trim.lower)` means “decode a JSON array, then decode each item with the nested string codec.”

`default(...)` can appear anywhere in the chain, but putting it at the end reads best because it describes the final fallback value after the normalization pipeline has been fully defined.

### Custom Codecs 

You can provide your own codecs. In practice, `props` does not require that every value come from Rocket’s built-in registry. Any value that implements the codec contract can be returned from the `props` object.

```
export type Codec<T> = {
  decode(value: unknown): T
  encode(value: T): string
}
```

Use `createCodec(...)` when you want Rocket to turn a plain decode/encode pair into a prop codec that behaves like the built-in ones.

`decode(value: unknown)` uses `unknown` on purpose. Raw custom-element attributes do arrive as strings, but Rocket also reuses codec decode paths for missing values, nested object and array members, and already-materialized JavaScript values. Using `unknown` keeps the contract honest: a codec should be able to normalize whatever input Rocket hands it, not just a string from HTML.

If a codec `decode(...)` throws, Rocket calls `console.warn(...)` and falls back to that codec’s default value instead. That applies to built-in codecs and to custom codecs returned from `createCodec(...)` or provided directly in `props`.

Most custom codecs should wrap an existing codec rather than starting from scratch. That lets you keep Rocket’s normal defaulting and attribute-reflection behavior while only changing the part that is specific to your domain.

```
import { createCodec } from '/bundles/datastar-pro.js'

let myCodec = createCodec({
  decode(value) {
    // normalize input
  },
  encode(value) {
    // reflect back to attribute text
  },
})
```

```
import { createCodec, rocket } from '/bundles/datastar-pro.js'

let percent = createCodec({
    decode(value) {
      let text = String(value ?? '').trim().replace(/%$/, '')
      let number = Number.parseFloat(text)
      return Number.isFinite(number)
        ? Math.max(0, Math.min(100, number))
        : 0
    },
    encode(value) {
      return String(Math.max(0, Math.min(100, value)))
    },
})

rocket('demo-meter', {
  props: ({ string }) => ({
    value: percent.default(50),
    label: string.trim.default('Progress'),
  }),
})
```

If you want the exported types, Rocket exposes `Codec` and `CodecRegistry` from the public module entrypoint.

### Codec Tables 

Rocket ships these codecs in the `props` registry.

CodecDecoded typeTypical inputTypical uses `string``string``" hello "`Text props, labels, ids, classes, case-normalized names.`number``number``"42"`Ranges, dimensions, timing, scores, percentages.`bool``boolean``""`, `"true"`, `"1"`Feature flags and toggles.`date``Date``"2026-03-18T12:00:00.000Z"`Timestamps and schedule props.`json``any``'{"items":[1,2,3]}'`Structured JSON payloads.`js``any``"{ foo: 1, bar: [2, 3] }"`JS-like object literals when strict JSON is inconvenient.`bin``Uint8Array`text-like binary payloadBinary or byte-oriented props.`array(codec)``T[]``'["a","b"]'`Lists of values with per-item normalization.`array(codecA, codecB, ...)`Tuple`'["en",10,true]'`Fixed-length ordered values.`object(shape)`Typed object`'{"x":10,"y":20}'`Named structured props.`oneOf(...)`Union`"primary"`Enums and constrained variants.

### `string` 

`string` is the most composable codec. It is useful on its own, but it also acts as a normalization pipeline any time a value eventually needs to become text.

Without an explicit `.default(...)`, the zero value is `""`.

MemberEffectExample `.trim`Removes surrounding whitespace.`" Ada "` becomes `"Ada"`.`.upper`Uppercases the string.`"ion"` becomes `"ION"`.`.lower`Lowercases the string.`"Rocket"` becomes `"rocket"`.`.kebab`Converts to kebab case.`"Demo Button"` becomes `"demo-button"`.`.camel`Converts to camel case.`"rocket button"` becomes `"rocketButton"`.`.snake`Converts to snake case.`"Rocket Button"` becomes `"rocket_button"`.`.pascal`Converts to Pascal case.`"rocket button"` becomes `"RocketButton"`.`.title`Title-cases each word.`"hello world"` becomes `"Hello World"`.`.prefix(value)`Adds a prefix if missing.`"42"` with `prefix('#')` becomes `"#42"`.`.suffix(value)`Adds a suffix if missing.`"24"` with `suffix('px')` becomes `"24px"`.`.maxLength(n)`Truncates to `n` characters.`"abcdef"` with `maxLength(4)` becomes `"abcd"`.`.default(value)`Supplies a fallback string.Missing values can become `"Anonymous"`.

```
props: ({ string }) => ({
  slug: string.trim.lower.kebab.maxLength(48),
  cssSize: string.trim.suffix('px').default('16px'),
  title: string.trim.title.default('Untitled'),
})
```

### `number` 

`number` turns a prop into a numeric API and lets you enforce range, rounding, snapping, and remapping rules right in the prop definition.

Without an explicit `.default(...)`, the zero value is `0`.

MemberEffectExample `.min(value)`Enforces a lower bound.`-4` with `min(0)` becomes `0`.`.max(value)`Enforces an upper bound.`120` with `max(100)` becomes `100`.`.clamp(min, max)`Applies both bounds.`120` with `clamp(0, 100)` becomes `100`.`.step(step, base?)`Snaps to the nearest increment.`13` with `step(5)` becomes `15`.`.round`Rounds to the nearest integer.`3.6` becomes `4`.`.ceil(decimals?)`Rounds up with optional decimal precision.`1.231` with `ceil(2)` becomes `1.24`.`.floor(decimals?)`Rounds down with optional decimal precision.`1.239` with `floor(2)` becomes `1.23`.`.fit(inMin, inMax, outMin, outMax, clamped?, rounded?)`Maps one numeric range into another.`50` from `0-100` into `0-1` becomes `0.5`.`.default(value)`Supplies a fallback number.Missing values can become `0` or `1`.

```
props: ({ number }) => ({
  width: number.min(0).default(320),
  opacity: number.clamp(0, 1).ceil(2).default(1),
  progress: number.clamp(0, 100).step(5),
  normalizedX: number.fit(0, 1920, 0, 1, true, false),
})
```

### `bool` 

`bool` decodes common truthy attribute forms into a boolean. Empty-string attributes such as `<demo-dialog open>` decode to `true`.

Without an explicit `.default(...)`, the zero value is `false`.

MemberEffectNotes `.default(value)`Supplies the fallback boolean.`true`, `false`, or a factory function.

```
props: ({ bool }) => ({
  open: bool,
  disabled: bool,
  elevated: bool.default(true),
})
```

### `date` 

`date` decodes a prop into a `Date`. Invalid input falls back to a valid date object rather than leaving the component with an unusable value.

Without an explicit `.default(...)`, the zero value is a fresh valid `Date` created at decode time.

MemberEffectNotes `.default(value)`Supplies the fallback date.Prefer a factory like `() => new Date()` to create a fresh timestamp per instance.

```
props: ({ date }) => ({
  startAt: date.default(() => new Date()),
  endAt: date.default(() => new Date(Date.now() + 60_000)),
})
```

### `json` 

`json` parses JSON text and clones structured values so instances do not share mutable default objects by accident.

Without an explicit `.default(...)`, the zero value is an empty object .

MemberEffectTypical use `.default(value)`Supplies a fallback object or array.Payloads, chart series, settings blobs, filter state.

```
props: ({ json }) => ({
  series: json.default(() => []),
  options: json.default(() => ({ stacked: false, legend: true })),
})
```

### `js` 

`js` is similar to `json` but accepts JavaScript-like object syntax, not just strict JSON. Use it when consumers will hand-author complex literals in HTML and you want a more forgiving parser.

Without an explicit `.default(...)`, the zero value is an empty object .

MemberEffectTypical use `.default(value)`Supplies a fallback object or array.Config literals that are easier to write without quoted keys.

```
props: ({ js }) => ({
  config: js.default(() => ({
    scale: 1,
    axis: { x: true, y: true },
  })),
})
```

### `bin` 

`bin` decodes base64 string input into `Uint8Array` and encodes bytes back into base64. Use it when the component’s natural public API is binary rather than textual.

Without an explicit `.default(...)`, the zero value is an empty `Uint8Array`.

MemberEffectTypical use `.default(value)`Supplies fallback bytes.Byte buffers, encoded data, binary previews.

```
props: ({ bin }) => ({
  payload: bin,
})
```

### `array` 

`array` has two forms. With one nested codec it creates a homogeneous array. With multiple codecs it creates a tuple.

Without an explicit `.default(...)`, `array(codec)` defaults to `[]`. Tuple forms default each missing slot from that slot codec’s own default or zero value.

FormDecoded typeWhat it means `array(codec)``T[]`Every item is decoded with the same codec.`array(codecA, codecB, codecC)`TupleEach position has its own codec and default behavior.

```
props: ({ array, string, number, bool }) => ({
  tags: array(string.trim.lower),
  point: array(number, number),
  localeSpec: array(
    string.lower.default('en'),
    number.min(1).default(1),
    bool,
  ).default(() => ['en', 1, false]),
})
```

Homogeneous arrays are ideal for tags, ids, and numeric series. Tuples are better when position matters, such as coordinates, breakpoints, or fixed parser options.

### `object` 

`object(shape)` builds a typed nested object. Each field has its own codec, so you can mix strings, numbers, booleans, arrays, and even nested objects inside a single prop.

Without an explicit `.default(...)`, each field falls back to that field codec’s own default or zero value.

MemberEffectNotes `object(shape)`Creates a fixed-key decoded object.Missing nested fields use their nested codec defaults when present.`.default(value)`Supplies a fallback object.Prefer a factory to create per-instance objects.

```
props: ({ object, string, number, bool, array }) => ({
  profile: object({
    id: string.trim,
    name: string.trim.default('Anonymous'),
    age: number.min(0),
    admin: bool,
    tags: array(string.trim.lower),
  }).default(() => ({
    id: '',
    name: 'Anonymous',
    age: 0,
    admin: false,
    tags: [],
  })),
})
```

### `oneOf` 

`oneOf` constrains a prop to a known set of allowed values. You can pass literal values, codecs, or both.

Without an explicit `.default(...)`, the zero value is the first allowed entry.

FormTypical usesBehavior `oneOf('a', 'b', 'c')`Enums and variant names.Returns the matching literal or the first/default entry.`oneOf(codecA, codecB)`Union-like decoding.Tries each codec in order until one succeeds.`.default(value)`Explicit fallback.Overrides the implicit “first option wins” fallback.

```
props: ({ oneOf, string, number }) => ({
  tone: oneOf('neutral', 'info', 'success', 'warning', 'danger').default('neutral'),
  alignment: oneOf('start', 'center', 'end').default('start'),
  flexibleValue: oneOf(string.trim, number.round),
})
```

## Setup and Actions 

`setup` is where Rocket components become stateful when that state does not depend on rendered refs. The context object gives you a focused set of hooks to create local Datastar-backed state and wire browser behavior.

### Setup Context 

HelperWhat it doesWhy it helps web components `props`The normalized prop values for the current instance.Keeps setup code on the same decoded inputs that render uses, without a second function argument.`$$`Creates mutable instance-local state and exposes it on `$$.name`.Gives each component instance its own Datastar-backed state bucket with a property-style API that mirrors template `$$name` access.`effect(fn)`Runs a reactive side effect and tracks cleanup.Ideal for timers, subscriptions, and imperative DOM/library sync.`apply(root, merge?)`Runs Datastar apply on a root.Useful when third-party code injects DOM that needs Datastar activation.`cleanup(fn)`Registers disconnect cleanup.Prevents leaked timers, observers, and library instances.`$`Reads and writes the global Datastar signal store.Useful when setup needs shared app state instead of component-local Rocket state.`actions`Calls Datastar global actions from setup code.Useful when component setup needs the same global helpers available to `@action(...)` expressions.`action(name, fn)`Registers a local action callable from rendered markup.Lets event handlers target the current component instance instead of global actions.`observeProps(fn, ...propNames)`Responds to prop changes after decoding.Separates prop-driven imperative work from full rerenders.`overrideProp(name, getter?, setter?)`Wraps a declared prop’s default host accessor for this instance.Useful when a public prop must read from or write through a live inner control.`defineHostProp(name, descriptor)`Defines a host-only property or method on this instance.Useful for native-like host APIs that are not Rocket props, such as `files` or imperative methods.`render(overrides, ...args)`Reruns the component render function from setup code.Lets async or imperative work trigger a render with explicit trailing args.`host`The current custom element instance.Gives access to attributes, classes, observers, focus, and shadow APIs.

```
rocket('demo-copy-button', {
  props: ({ string, number }) => ({
    text: string.default('Copy me'),
    resetMs: number.min(100).default(1200),
  }),
  setup: ({ $$, $, action, actions, cleanup, props }) => {
    $$.copied = false
    $$.label = () => ($$.copied ? 'Copied' : 'Copy')
    $$.resetMsLabel = actions.intl(
      'number',
      props.resetMs,
      { maximumFractionDigits: 0 },
      'en-US',
    )
    let timerId = 0

    action('copy', async () => {
      await navigator.clipboard.writeText(props.text)
      $$.copied = true
      if ($.analyticsEnabled !== false) {
        $.lastCopiedText = props.text
      }
      clearTimeout(timerId)
      timerId = window.setTimeout(() => {
        $$.copied = false
      }, props.resetMs)
    })

    cleanup(() => clearTimeout(timerId))
  },
  render: ({ html, props: { text } }) => {
    return html`
      <button data-on:click="@copy()">
        <span data-text="$$label"></span>
        <small>${text} ($$resetMsLabel ms)</small>
      </button>
    `
  },
})
```

Local actions are optional. Prefer plain Datastar expressions like `data-on:click="$$count += 1"` when local state changes are simple, and prefer page-owned state when the behavior belongs to the surrounding demo or app. Reach for `action(name, fn)` when the markup needs a named imperative entry point.

`$$` is the setup alias for Rocket-local signals. In practice that means `$$.count = 0` creates `$$count`, which templates can read, and also enables `$$.count` and `$$.count += 1` inside `setup`.

Assigning a function to `$$.name` creates a local computed signal, so `$$.label = () => $$.count + 1` is the shorthand form of a derived value. Use that form for derived local state.

`actions` exposes the global Datastar action registry inside `setup`. Use it when setup code needs the same helpers available to declarative expressions like `@intl(...)` or `@clipboard(...)`, without re-registering them as local Rocket actions.

`$` exposes the shared Datastar signal root inside `setup`. Use it when a component needs to coordinate with application-level state instead of only reading and writing Rocket’s instance-local signals.

Local refs created with `data-ref:name` are exposed on `onFirstRender({ refs })` as `refs.name`. They are populated during the Datastar apply pass, so they are intentionally not part of `setup(...)`. They are Rocket refs, not Rocket signals, so they do not appear on `$$`.

### Host Accessor Overrides 

Most components should stop at plain `props`. Rocket already gives each prop a host accessor plus decoded values, attribute reflection, upgrade replay, and `observeProps()` updates.

Use `overrideProp(name, getter?, setter?)` when that default accessor is not the right host API. Typical cases are native-like form wrappers where `host.value` or `host.checked` should mirror a live inner control instead of only returning the last decoded prop value.

Use `defineHostProp(name, descriptor)` when the member is not a Rocket prop at all. Good examples are read-only host properties like `files` or imperative host methods like `start()` and `stop()`.

Do not use accessor overrides for internal state. If outside code should not read or set it through the host element, keep it in `$$`. Also do not move every prop to an own-property just to be safe. Prototype accessors remain the default fast path.

The override helpers are available during `setup`:

```
overrideProp<Name extends keyof Props & string>(
  name: Name,
  getter?: (getDefault: () => Props[Name]) => any,
  setter?: (value: any, setDefault: (value: Props[Name]) => void) => void,
): void

defineHostProp(name: string, descriptor: PropertyDescriptor): void
```

If you omit `getter`, Rocket uses `getDefault()`. If you omit `setter`, Rocket uses `setDefault(value)`. That keeps the common case short when you only need one side customized.

```
rocket('demo-prop-normalize', {
  props: ({ string }) => ({
    value: string.default(''),
  }),
  setup: ({ overrideProp }) => {
    overrideProp('value', undefined, (value, setDefault) => {
      setDefault(String(value ?? '').trim())
    })
  },
})
```

Setup runs before Rocket’s initial render and Datastar apply pass. That means `data-ref:*` refs like `refs.input` or `refs.select` do not exist yet during the first synchronous line of `setup`.

If an override depends on rendered refs, put that wiring in `onFirstRender(...)`. It receives the setup context plus `refs`, so ref-backed code still has access to `$$`, props, cleanup, host helpers, and the rendered DOM refs.

```
rocket('demo-input-bridge', {
  props: ({ string }) => ({
    value: string.default(''),
  }),
  render: ({ html, props: { value } }) => html`
    <input data-ref:input value="${value}">
  `,
  onFirstRender: ({ overrideProp, refs }) => {
    overrideProp(
      'value',
      (getDefault) => refs.input?.value ?? getDefault(),
      (value, setDefault) => {
        const next = String(value ?? '')
        if (refs.input && refs.input.value !== next) refs.input.value = next
        setDefault(next)
      },
    )
  },
})
```

```
rocket('demo-host-methods', {
  setup: ({ defineHostProp }) => {
    defineHostProp('version', {
      get() {
        return '1'
      },
    })
    defineHostProp('reset', {
      value() {
        console.log('reset')
      },
    })
  },
})
```

### Watching Attribute Changes 

`observeProps` is useful when prop changes should drive targeted imperative work. The callback receives the full normalized props object plus a `changes` object containing only the props that changed. If you omit `propNames`, `observeProps(fn)` watches all props.

```
rocket('demo-video-frame', {
  props: ({ string, number }) => ({
    src: string.trim,
    currentTime: number.min(0),
  }),
  mode: 'light',
  renderOnPropChange: false,
  onFirstRender: ({ refs, observeProps }) => {
    observeProps((props, changes) => {
      if (!(refs.video instanceof HTMLVideoElement)) {
        return
      }
      if ('src' in changes) {
        refs.video.src = props.src
      }
      if ('currentTime' in changes) {
        refs.video.currentTime = props.currentTime
      }
    })
  },
  render: ({ html, props: { src } }) => {
    return html`
        <video data-ref:video controls src="${src}"></video>
    `
  },
})
```

## Rendering and Scoping 

Rocket rendering is Datastar-aware. The output of `render` is not just a string template. Rocket parses it, converts it into DOM, rewrites local signal references, morphs the mounted subtree, and then applies Datastar behavior to the result.

### Render Contract 

The render context is:

FieldWhat it gives youWhy it exists `html`An HTML tagged template that returns a fragment.Safely constructs HTML while supporting node composition and Datastar rewriting.`svg`An SVG tagged template that returns SVG nodes.Lets a component render SVG without manual namespace handling.`props`The normalized prop values.Keeps markup based on already-decoded component inputs.`host`The custom element instance.Allows render decisions based on host state, slots, or attributes.

You should think of `render` as the component’s declarative DOM shape, not as an all-purpose setup hook. Create signals and effects in `setup`. Use `render` to express what the DOM should look like for the current props and local state.

If a light-DOM component needs to keep host-provided children, return `<slot>` markers where those children should go. That is not native slotting. In light mode Rocket uses `<slot>` as a projection marker because the browser only performs real slot distribution in shadow DOM. Rocket replaces those slot nodes with the original host children before morphing the host subtree.

### Render Example: Counter 

This is the smallest useful Rocket render pattern: typed props define the public API, `setup` creates local state because no refs are needed, and the returned HTML reads and writes local signals with standard Datastar attributes.

```
rocket('demo-stepper', {
  mode: 'light',
  props: ({ number, string }) => ({
    start: number.min(0),
    step: number.min(1).default(1),
    label: string.trim.default('Count'),
  }),
  setup: ({ $$, props }) => {
    $$.count = props.start
  },
  render: ({ html, props: { label, step } }) => {
    return html`
      <section class="stack gap-2">
        <h3>${label}</h3>
        <div class="row gap-2">
          <button data-on:click="$$count -= ${step}" data-attr:disabled="$$count <= 0">-</button>
          <output data-text="$$count"></output>
          <button data-on:click="$$count += ${step}">+</button>
        </div>
      </section>
    `
  },
})
```

The important detail is that the event handlers and bindings are just Datastar attributes. Rocket does not invent a second templating language for markup. It only scopes component state and packages the result as a reusable custom element.

### Render Example: List Rendering 

The `html` helper accepts composed nodes and iterables, so complex render output can stay declarative without string concatenation.

```
rocket('demo-nav-list', {
  props: ({ array, object, string }) => ({
    items: array(object({
      href: string.trim.default('#'),
      label: string.trim.default('Untitled'),
    })),
    title: string.trim.default('Navigation'),
  }),
  render: ({ html, props: { items, title } }) => {
    return html`
      <nav aria-label="${title}">
        <h3>${title}</h3>
        <ul>
          ${items.map((item) => html`
            <li>
              <a href="${item.href}">${item.label}</a>
            </li>
          `)}
        </ul>
      </nav>
    `
  },
})
```

That matters for web components because real components often render lists of nested child nodes. Rocket lets you return fragments from inner templates instead of dropping down to manual DOM creation.

### Render Example: SVG 

Use `svg` when the output is naturally vector-based. Rocket handles the SVG namespace and still rewrites Datastar expressions inside the returned nodes.

```
rocket('demo-meter-ring', {
  props: ({ number, string }) => ({
    value: number.clamp(0, 100),
    stroke: string.default('#0f172a'),
  }),
  render: ({ html, svg, props: { value, stroke } }) => {
    const circumference = 2 * Math.PI * 28

    return html`
      <figure class="stack gap-2">
        ${svg`
          <svg viewBox="0 0 64 64" width="64" height="64" aria-hidden="true">
            <circle cx="32" cy="32" r="28" fill="none" stroke="#e5e7eb" stroke-width="8"></circle>
            <circle
              cx="32"
              cy="32"
              r="28"
              fill="none"
              stroke="${stroke}"
              stroke-width="8"
              stroke-dasharray="${circumference}"
              stroke-dashoffset="${circumference - (value / 100) * circumference}"
              transform="rotate(-90 32 32)"></circle>
          </svg>
        `}
        <figcaption>${value}%</figcaption>
      </figure>
    `
  },
})
```

### Conditional Rendering 

Rocket supports structural conditionals inside `render` with `<template data-if>`, `data-else-if`, and `data-else`.

```
rocket('demo-status', {
  mode: 'light',
  setup: ({ $$ }) => {
    $$.step = 0
  },
  render: ({ html }) => html`
    <div class="stack gap-2">
      <button type="button" data-on:click="$$step = ($$step + 1) % 3">Next</button>

      <template data-if="$$step === 0">
        <p>Idle</p>
      </template>
      <template data-else-if="$$step === 1">
        <p>Loading</p>
      </template>
      <template data-else>
        <p>Ready</p>
      </template>
    </div>
  `,
})
```

Only one branch in the chain is mounted at a time. Inactive branches are not present in the live DOM. Switching branches unmounts the old branch and mounts a fresh new one.

Conditionals are owned by the Rocket runtime rather than by normal Datastar attribute plugins. That lets Rocket defer `$$` rewriting until the selected branch is actually mounted. If you need an element to stay mounted and only change visibility, use `data-show` instead.

### Loop Rendering 

Rocket also supports structural list rendering with `<template data-for>`.

```
rocket('demo-letter-list', {
  mode: 'light',
  setup: ({ $$ }) => {
    $$.letters = ['A', 'B', 'C']
  },
  render: ({ html }) => html`
    <ul>
      <template data-for="letter, row in $$letters">
        <li>
          <strong data-text="row + 1"></strong>
          <span data-text="letter"></span>
        </li>
      </template>
    </ul>
  `,
})
```

`data-for` accepts any Datastar expression that evaluates to an iterable and supports exactly three shapes:

FormAccepted ShapeLoop Locals `data-for="$$letters"`Bare iterable expression with default aliases.`item`, `i``data-for="letter in $$letters"`Custom item alias with the default index alias.`letter`, `i``data-for="letter, row in $$letters"`Custom item alias and custom index alias.`letter`, `row`

The source expression can be any iterable Datastar expression, so forms like `data-for="letter, row in $$letters.filter(Boolean)"` and `data-for="$page.items"` are both valid.

Rocket does not support an index-only form like `data-for=", row in $$letters"`. If you want a custom index alias, you must also provide an item alias.

Those loop locals are only available inside Datastar expressions in the repeated subtree. That means attributes like `data-text="item"`, `data-text="letter"`, and `data-class:active="i === 0"` work inside the loop body, while normal component locals like `$$selected` and global signals like `$page` keep their existing meaning outside those aliases.

Structural templates in Rocket, including conditionals and `data-for`, clone their selected `<template>` content into document fragments and then hand the resulting nodes back to Rocket’s normal patch/morph path. When the source list changes, Rocket keeps row slots by position and updates the current `item`/`i` bindings for each slot. If you reorder the source, Rocket does not preserve item identity across rows in this version.

If you author literal Datastar expressions that contain `${...}` inside Rocket example files, Biome can flag them with `lint/suspicious/noTemplateCurlyInString` even though they are intentional. This is the Biome config override this repo uses for Rocket example sources:

```
{
  "overrides": [
    {
      "includes": ["site/shared/static/rocket/**/*.js"],
      "linter": {
        "rules": {
          "suspicious": {
            "noTemplateCurlyInString": "off"
          }
        }
      }
    }
  ]
}
```

### Rocket Scope Rewriting 

Inside rendered Datastar expressions, Rocket rewrites `$$name` to an instance-specific signal path under `$._rocket`. That is what gives every component instance isolated local state while still using Datastar’s global signal store.

The instance segment comes from the host element’s `id` when one is present. Rocket normalizes that id into a path-safe identifier for Datastar expressions. If the element has no `id`, Rocket generates a sequential fallback instance id instead.

```
// You write this in render():
html`
  <button data-on:click="$$count += 1"></button>
  <span data-text="$$count"></span>
`

// For <demo-counter id="inventory-panel">, Rocket rewrites it as:
<button data-on:click="_rocket.demo_counter.inventory_panel.count += 1"></button>
<span data-text="_rocket.demo_counter.inventory_panel.count"></span>
```

That rewriting is what makes Rocket practical for reusable web components. You can drop ten instances of the same component into a page and each one gets separate local state without naming collisions. In normal component code you should still write `$$count`, not the rewritten `_rocket...` path directly.

### `__root` 

Use the `__root` modifier when a Rocket component needs to leave a signal-name attribute in the outer page scope instead of rewriting it into the component’s private `$._rocket...` path.

This is mainly for authored host children inside `open` or `closed` components. By default Rocket rescopes those children so `data-bind:name` inside a component instance becomes something like `data-bind:_rocket.my_component.id.name`. That is usually correct for component-local behavior, but it is wrong when the child should keep talking to a page-level signal like `$name`.

```
<demo-fieldset>
  <demo-input data-bind:name__root></demo-input>
</demo-fieldset>
```

In that example, Rocket strips `__root` and leaves the binding as `data-bind:name` instead of rewriting it into the component scope.

Use `__root` sparingly. It is an escape hatch for wrapper-style components where host children should stay connected to outer-page Datastar signals. Do not use it for normal component internals, and do not use it when the signal should actually be instance-local.

For keyed signal-name attributes, put the modifier on the key segment: `data-bind:name__root`, `data-computed:total__root`, `data-indicator:loading__root`, `data-ref:input__root`. This keyed form is the right choice for authored host children because Datastar can still parse the base attribute shape before Rocket rewrites it.

Rocket currently applies `__root` to these signal-name attribute families:

- `data-bind` and `data-bind:*`
- `data-computed:*`
- `data-indicator` and `data-indicator:*`
- `data-ref` and `data-ref:*`

It does not currently affect every `data-*` attribute. In particular, attributes like `data-attr:*`, `data-text`, and Rocket’s own internal host/ref bookkeeping are separate concerns.

## Examples 

The best live references in the repo are the Rocket examples: [Copy Button](https://data-star.dev/examples/rocket_copy_button), [Counter](https://data-star.dev/examples/rocket_counter), [Flow](https://data-star.dev/examples/rocket_flow), [Letter Stream](https://data-star.dev/examples/rocket_letter_stream), [QR Code](https://data-star.dev/examples/rocket_qr_code), [Starfield](https://data-star.dev/examples/rocket_starfield), and [Virtual Scroll](https://data-star.dev/examples/rocket_virtual_scroll).

Use this page as the API reference and those pages as the behavioral reference. Between the two, you should be able to define a typed, reactive, reusable web component without falling back to ad hoc component wiring.

# SSE Events

Responses to [backend actions](https://data-star.dev/reference/actions#backend-actions) with a content type of `text/event-stream` can contain zero or more Datastar [SSE events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events).

> The backend [SDKs](https://data-star.dev/reference/sdks) can handle the formatting of SSE events for you, or you can format them yourself.

## Event Types 

### `datastar-patch-elements` 

Patches one or more elements in the DOM. By default, Datastar morphs elements by matching top-level elements based on their ID.

```
event: datastar-patch-elements
data: elements <div id="foo">Hello world!</div>

```

In the example above, the element `<div id="foo">Hello world!</div>` will be morphed into the target element with ID `foo`. Note that SSE events must be followed by two newline characters.

> Be sure to place IDs on top-level elements to be morphed, as well as on elements within them that you’d like to preserve state on (event listeners, CSS transitions, etc.).

Morphing elements within SVG elements requires special handling due to XML namespaces. See the [SVG morphing example](https://data-star.dev/examples/svg_morphing).

Additional `data` lines can be added to the response to override the default behavior.

KeyDescription `data: selector #foo`Selects the target element of the patch using a CSS selector. Not required when using the `outer` or `replace` modes.`data: mode outer`Morphs the outer HTML of the elements. This is the default (and recommended) mode.`data: mode inner`Morphs the inner HTML of the elements.`data: mode replace`Replaces the outer HTML of the elements.`data: mode prepend`Prepends the elements to the target’s children.`data: mode append`Appends the elements to the target’s children.`data: mode before`Inserts the elements before the target as siblings.`data: mode after`Inserts the elements after the target as siblings.`data: mode remove`Removes the target elements from DOM.`data: namespace svg`Patch elements into the DOM using an `svg` namespace.`data: namespace mathml`Patch elements into the DOM using a `mathml` namespace.`data: useViewTransition true`Whether to use view transitions when patching elements. Defaults to `false`.`data: elements`The HTML elements to patch.

```
event: datastar-patch-elements
data: elements <div id="foo">Hello world!</div>

```

Elements can be removed using the `remove` mode and providing a `selector`.

```
event: datastar-patch-elements
data: selector #foo
data: mode remove

```

Elements can span multiple lines. Sample output showing non-default options:

```
event: datastar-patch-elements
data: selector #foo
data: mode inner
data: useViewTransition true
data: elements <div>
data: elements        Hello world!
data: elements </div>

```

Elements can be patched using `svg` and `mathml` namespaces by specifying the `namespace` data line.

```
event: datastar-patch-elements
data: namespace svg
data: elements <circle id="circle" cx="100" r="50" cy="75"></circle>

```

### `datastar-patch-signals` 

Patches signals into the existing signals on the page. The `onlyIfMissing` line determines whether to update each signal with the new value only if a signal with that name does not yet exist. The `signals` line should be a valid `data-signals` attribute.

```
event: datastar-patch-signals
data: signals {foo: 1, bar: 2}

```

Signals can be removed by setting their values to `null`.

```
event: datastar-patch-signals
data: signals {foo: null, bar: null}

```

Sample output showing non-default options:

```
event: datastar-patch-signals
data: onlyIfMissing true
data: signals {foo: 1, bar: 2}

```

# SDKs

Datastar provides backend SDKs that can (optionally) simplify the process of generating [SSE events](https://data-star.dev/reference/sse_events) specific to Datastar.

> If you’d like to contribute an SDK, please follow the [Contribution Guidelines](https://github.com/starfederation/datastar/blob/main/CONTRIBUTING.md#sdks).

## Clojure 

A Clojure SDK as well as helper libraries and adapter implementations.

*Maintainer: [Jeremy Schoffen](https://github.com/JeremS)*

[Clojure SDK & examples](https://github.com/starfederation/datastar-clojure)

## C# 

A C# (.NET) SDK for working with Datastar.

*Maintainer: [Greg H](https://github.com/SpiralOSS)*  
*Contributors: [Ryan Riley](https://github.com/panesofglass)*

[C# (.NET) SDK & examples](https://github.com/starfederation/datastar-dotnet/)

## Go 

A Go SDK for working with Datastar.

*Maintainer: [Delaney Gillilan](https://github.com/delaneyj)*

*Other examples: [1 App 5 Stacks ported to Go+Templ+Datastar](https://github.com/delaneyj/1a5s-datastar)*

[Go SDK & examples](https://github.com/starfederation/datastar-go)

## Haskell 

A Haskell SDK for working with Datastar.

*Maintainer: [Carlo Hamalainen](https://github.com/carlohamalainen)*

[Haskell SDK & examples](https://github.com/starfederation/datastar-haskell)

## Java 

A Java SDK for working with Datastar.

*Maintainer: [mailq](https://github.com/mailq)*  
*Contributors: [Peter Humulock](https://github.com/rphumulock), [Tom D.](https://github.com/anastygnome)*

[Java SDK & examples](https://github.com/starfederation/datastar-java)

## Kotlin 

A Kotlin SDK for working with Datastar.

*Maintainer: [GuillaumeTaffin](https://github.com/GuillaumeTaffin)*

[Kotlin SDK & examples](https://github.com/starfederation/datastar-kotlin)

## PHP 

A PHP SDK for working with Datastar.

*Maintainer: [Ben Croker](https://github.com/bencroker)*

[PHP SDK & examples](https://github.com/starfederation/datastar-php)

### Craft CMS 

Integrates the Datastar framework with [Craft CMS](https://craftcms.com/), allowing you to create reactive frontends driven by Twig templates.

*Maintainer: [Ben Croker](https://github.com/bencroker) ([PutYourLightsOn](https://putyourlightson.com/))*

[Craft CMS plugin](https://putyourlightson.com/plugins/datastar)

[Datastar & Craft CMS demos](https://craftcms.data-star.dev/)

### Laravel 

Integrates the Datastar hypermedia framework with [Laravel](https://laravel.com/), allowing you to create reactive frontends driven by Blade views or controllers.

*Maintainer: [Ben Croker](https://github.com/bencroker) ([PutYourLightsOn](https://putyourlightson.com/))*

[Laravel package](https://github.com/putyourlightson/laravel-datastar)

## Python 

A Python SDK and a [PyPI package](https://pypi.org/project/datastar-py/) (including support for most popular frameworks).

*Maintainer: [Felix Ingram](https://github.com/lllama)*  
*Contributors: [Chase Sterling](https://github.com/gazpachoking)*

[Python SDK & examples](https://github.com/starfederation/datastar-python)

## Ruby 

A Ruby SDK for working with Datastar.

*Maintainer: [Ismael Celis](https://github.com/ismasan)*

[Ruby SDK & examples](https://github.com/starfederation/datastar-ruby)

## Rust 

A Rust SDK for working with Datastar.

*Maintainer: [Glen De Cauwsemaecker](https://github.com/glendc)*  
*Contributors: [Johnathan Stevers](https://github.com/jmstevers)*

[Rust SDK & examples](https://github.com/starfederation/datastar-rust)

### Rama 

Integrates Datastar with [Rama](https://ramaproxy.org/), a Rust-based HTTP proxy ([example](https://github.com/plabayo/rama/blob/main/examples/http_sse_datastar_hello.rs)).

*Maintainer: [Glen De Cauwsemaecker](https://github.com/glendc)*

[Rama module](https://ramaproxy.org/docs/rama/http/sse/datastar/index.html)

## Scala 

### ZIO HTTP 

Integrates the Datastar hypermedia framework with [ZIO HTTP](https://ziohttp.com/), a Scala framework.

*Maintainer: [Nabil Abdel-Hafeez](https://github.com/987Nabil)*

[ZIO HTTP integration](https://ziohttp.com/reference/datastar-sdk/)

## TypeScript 

A TypeScript SDK with support for Node.js, Deno, and Bun.

*Maintainer: [Edu Wass](https://github.com/eduwass)*  
*Contributors: [Patrick Marchand](https://github.com/Superpat)*

[TypeScript SDK & examples](https://github.com/starfederation/datastar-typescript)

### PocketPages 

Integrates the Datastar framework with [PocketPages](https://pocketpages.dev/).

[PocketPages plugin](https://github.com/benallfree/pocketpages/tree/main/packages/plugins/datastar)

## Unison 

A Unison SDK for working with Datastar.

*Maintainer: [Kaushik Chakraborty](https://github.com/kaychaks)*

[Unison SDK & examples](https://share.unison-lang.org/@kaychaks/datastar-unison)

# Security

[Datastar expressions](https://data-star.dev/guide/datastar_expressions) are strings that are evaluated in a sandboxed context. This means you can use JavaScript in Datastar expressions.

## Escape User Input 

The golden rule of security is to never trust user input. This is especially true when using Datastar expressions, which can execute arbitrary JavaScript. When using Datastar expressions, you should always escape user input. This helps prevent, among other issues, Cross-Site Scripting (XSS) attacks.

## Avoid Sensitive Data 

Keep in mind that signal values are visible in the source code in plain text, and can be modified by the user before being sent in requests. For this reason, you should avoid leaking sensitive data in signals and always implement backend validation.

## Ignore Unsafe Input 

If, for some reason, you cannot escape unsafe user input, you should ignore it using the [`data-ignore`](https://data-star.dev/reference/attributes#data-ignore) attribute. This tells Datastar to ignore an element and its descendants when processing DOM nodes.

## Content Security Policy 

When using a [Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP) (CSP), `unsafe-eval` must be allowed for scripts, since Datastar evaluates expressions using a [`Function()` constructor](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Function/Function).

```
<meta http-equiv="Content-Security-Policy" 
    content="script-src 'self' 'unsafe-eval';"
>
```
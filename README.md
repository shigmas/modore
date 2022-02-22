# BACnet library in go.

It's not really organized yet.

bacnet is the main library. See connection_test.go for a Who Is example, which is the only thing implemented. APDU is "Application Protocol Data Unit", which is wrapped in a "Network Data Protocol Unit", or NPDU. It seems like the user cares about the application, and not the network. So, the creates a connection, and sends an apdu Who Is message on the client. It gives the connection a channel to receive responses.


Design:
At least npdu, and probably apdu should be moved to internal, and we can provide a public API for the objects.

## organization
### internal
This area deals mostly with the encoding and decoding of the NPDU and APDU. The parts that are required for external use are exported. These are the types that need to be filtered.

### pkg
This is broken into
 - bacnet: This generically named package is for BACnet types, like message classes and types. Since there are a few layers of types, the names are clear for their meanings (or at least the attempt was made). Since this needs to be encoded, this is imported by internal, so the types are just interfaces. This package also creates the messages
 - transport: This has the networking related types. This could be considered the entry point for the module.

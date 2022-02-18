# BACnet library in go.

It's not really organized yet.

bacnet is the main library. See connection_test.go for a Who Is example, which is the only thing implemented. APDU is "Application Protocol Data Unit", which is wrapped in a "Network Data Protocol Unit", or NPDU. It seems like the user cares about the application, and not the network. So, the creates a connection, and sends an apdu Who Is message on the client. It gives the connection a channel to receive responses.

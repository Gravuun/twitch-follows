# Who Do I Follow on Twitch

Who do I follow on Twitch is a website that simply asks the user to log in via the Twitch OIDC authorization code flow and then gets a list of the channels the user follows

## Why?

To play around with OIDC and Go

## Issues

I have a lot to still learn about in Go so there will be sub-optimal choices in this code that I can come back to and reason through.
Logout redirection is not working and upon navigating back to "/" and attempting to sign in there is no prompt to sign in (automatically signed in as user who just revoked access)

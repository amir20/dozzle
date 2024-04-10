---
title: Anonymous Analytics
---

# Data Collection of Analytics

Dozzle collects anonymous user configurations using a simple beacon written in Go. _Why?_ Dozzle is an open source project with no funding. As a result, there is no time to do user studies of Dozzle. Analytics is collected to prioritize features and fixes based on how people use Dozzle.

## Where is data stored

Dozzle's sends anonymous data to DigitalOcean which is written to a flat file for processing.

## Opting out

Dozzle analytics helps to prioritize features and spend time on most important features. If you do not want to be tracked at all, use `--no-analytics` flag or `DOZZLE_NO_ANALYTICS` env variable.

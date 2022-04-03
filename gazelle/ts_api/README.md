# ts_api

This folder contains a nodejs program that uses the TypeScript API to query for properties of
TypeScript applications in the users codebase.

It is designed to run as a subprocess of the Gazelle Go extension.
It speaks a json-over-stdio protocol, getting requests over stdin and responding over stdout.

This allows a single instance of this program to run alongside the Gazelle extension for the full
lifecycle of Gazelle, so that we don't pay the cost of starting it more than once.

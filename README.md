
# AWS CodePipeline Job Worker

A simple job worker for custom actions in AWS CodePipeline. Built to be easily extended to accomodate other custom
jobs not already implemented.

## How to Use
* Host anywhere
* [Set up your credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials)
* Run application 🎉

## How It Works
For each custom action you wish to set up, the application expects there to be a
`dispatcher` and a `job`. The `dispatcher` is responsible for implementing the methods
defined in the interface so that the shared `poller` can be used to query AWS CodePipeline
for new jobs. The `dispatcher` also sets up the job for a custom action and kicks it
off by calling `execute`. The main meat of the logic for the custom action should live in
the `job`, which is responsible for initializing itself, and reporting back the correct
status to AWS CodePipeline. See `actions/travis/dispatcher` and `actions/travis/job` for a
working example. The current overall flow of the application is as follows:

//TODO: create applicaiton diagram

<img src="https://cdn4.iconfinder.com/data/icons/under-construction-1/512/under-512.png" height="200px">



## TODO:
* Tons of refactoring, mainly in `job.rb`
  * Address TODOs
  * Refactor nested ifs and general error handling
* Add configuration support
* Add tests
* Improve documentation
* Add travis build
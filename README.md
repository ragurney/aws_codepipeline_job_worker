
# AWS CodePipeline Job Worker

A simple job worker for custom actions in AWS CodePipeline. Built to be easily extended to accomodate other custom
jobs not already implemented.

## How to Use
* Create your AWS CodePipeline and set up a custom action (included in this repo is a custom job configuration for Travis builds)
* Host `aws_codepipeline_job_worker` anywhere (laptop, EC2 instance, etc.).
* [Set up your credentials](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).
* Run application 🎉

## How It Works
### Structure
The structure of `actions/` should be as follows:
```
custom_action_1/
├── dispatcher.go
├── job.go
│
...
custom_action_N/
├── dispatcher.go
├── job.go
```
Each custom action is expected to have a `dispatcher` and a `job`.
`dispatcher` is responsible for implementing the methods
defined in the interface, which are responsible for starting the poller for that action type, as well as kicking off a new job. The main meat of the logic for the custom action should live in `job`, which is responsible for initializing itself, and reporting back the correct
status to AWS CodePipelin (see `pkg/actions/travis/dispatcher` and `pkg/actions/travis/job` for a working example.


### Application Flow

![worker_flow](https://user-images.githubusercontent.com/15261525/44610879-341b7780-a7b3-11e8-939b-1866f023fd44.png)

1. `main.go` fires off all dispatchers (e.x. `travis.NewDispatcher().Initialize()`).
1. The dispatcher starts a `poller` in a new thread for that particular custom action.
1. The poller polls CodePipeline intermittently, calling `PollForJobs`.
1. Once a job is available, CodePipeline will return the job.
1. `poller` will call it's `dispatcher`'s `Dispatch` method.
1. `dispatcher` will kick off the job in a new thread.
1. `job` will do all of the interaction with the third party service, such as submitting a build, polling for progress, updating CodePipeline with intermediate statuses/info such as an `ExecutionID`
1. Once the job is complete `job` will update CodePipeline a final time with the result status.

<img src="https://cdn4.iconfinder.com/data/icons/under-construction-1/512/under-512.png" height="100px">

* Tons of refactoring, mainly in `job.rb`
  * Address TODOs
  * Refactor nested ifs and general error handling
* Add configuration support
* Add tests
* Improve documentation
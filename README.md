# ![RealWorld Example App](logo.png)

> ### gin, gorm (Functional Programming) + Ginkgo/mock codebase containing real world examples (CRUD, auth, advanced patterns, etc) that adheres to the [RealWorld](https://github.com/gothinkster/realworld) spec and API.


This codebase was created to demonstrate a backend of a fully fledged fullstack application built with **Golang (Gin/Gorm) Functional Programming ** including CRUD operations, authentication, routing, pagination, and more.

We've gone to great lengths to adhere to the **Golang ** community style guides & best practices.

For more information on how to this works with other frontends/backends, head over to the [RealWorld](https://github.com/gothinkster/realworld) repo.


# How it works

It uses golang 1.23, and gin + gorm/sqlite  
It provides ability to handle concurrency with a small number of threads and scale with fewer hardware resources, with functional developemnt approach.


## Database
It uses embedded sqlite database for demonstration purposes, and save the trouble to install mongo db for the test run.
This code is also tested work with community version of mongodb. 


## Basic approach
The quality & architecture of this Conduit implementation reflect something similar to an early stage startup's MVP: functionally complete & stable, but not unnecessarily over-engineered.


## Project structure
```
- app - web layer which contains router function (router) and 
- handlers - handlers
- model - non-persistence tier data structures
- repository - includes entities, repositories and a support classes
- util - includes errors and misc sharable functions.
- middleware/security - security settings.
```
## Tests
1. Integration tests covers followings, 
- Repository test on customized repository impl
2. Unit tests utilize spock framework to mock the scenarios that not easily repeatable by Integration test
- app handlers
- routers
- Security


# Getting started
## Prerequisite
* install Golang 1.25 
* install ginkgo 
* install (uber) mockgen


Use go run to start this implementation of "Real World":

`go run main.go `

To test that it works, open a browser tab at http://localhost:8080/api/tags .  
Alternatively, you can run
```
curl http://localhost:8080/api/tags
```
or use "real world" postman collection at [here](https://github.com/gothinkster/realworld/blob/main/api/Conduit.postman_collection.json)

# Run test

The repository contains a lot of test cases to cover api, repository, and integration test.

`ginkgo -r -cover`

to view test report
`go tool cover --html=coverprofile.out`

# Help

Please fork and PR to improve the project.
Please leave suggestions or comment to the repo, or send to [alphawy\@hotmail.com](mailto:alphawy\@hotmail.com)


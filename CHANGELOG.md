# Changelog

All notable changes to this project will be documented in this file.

## (0.0.5) - 2018-05-08
### Fixed
- Multiple URLs in one database did not work
### Modified
- Forked "github.com/pmezard/go-difflib/difflib" to add "ignoreWhitespaces" option
### Added
- Implemented possibility to test sendMail function

## (0.0.4) - 2018-01-15
### Added
- Added support for Gitlab CI

## (0.0.3) - 2018-01-13
### Modified
- Switched from "github.com/sergi/go-diff/diffmatchpatch" to "github.com/pmezard/go-difflib/difflib"

## (0.0.2) - 2018-01-13
### Added
- Added tests
- Usage of Makefile to handle building and testing

## (0.0.1) - 2018-01-10
### Added
- Ability to set URL, Email To, Email From and TLS Host via Parameter
- First version downloads content and saves it into a local DB 
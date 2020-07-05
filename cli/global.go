package main

// <constants>
const existsErrMsg = `file '%s' already exists and --overwrite was not specified`
const configJSONErrMsg = `could not serialize config JSON: %s`
const resultJSONErrMsg = `could not serialize result JSON: %s`

// </constants>

// <global-variables>
//   <subset purpose="used by ‘cobra’">
var argConfigOutput bool
var argJSONOutput bool

// TODO var cfgFile string
//   </subset>

//   <subset purpose="used for passing values between ‘cobra’ methods">
var w Output
var log Output
var exitCode int
var cmdError error

//   </subset>
// </global-variables>

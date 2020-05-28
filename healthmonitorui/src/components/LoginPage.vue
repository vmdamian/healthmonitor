<template>
  <div class="hello">
    <h1>HealthMonitor Welcome Page</h1>
    <div v-if="!loginOK">
      <h2>Complete the following form to register a new user</h2>
      <input v-model="registerUsername" placeholder="username">
      <input v-model="registerPassword" placeholder="password">
      <button v-on:click="registerUser">Register</button>
    </div>
    <div v-if="!loginOK">
      <h2>Complete the following form to login with your existing user</h2>
      <input v-model="loginUsername" placeholder="username">
      <input v-model="loginPassword" placeholder="password">
      <button v-on:click="loginUser">Login</button>
    </div>
    <div v-if="loginOK">
      <h2>Requested device info</h2>
      <span> {{ deviceInfo }} </span>
      <h2>Device data plotted below</h2>
      <line-chart :data="temperatureData" :labels="labels" :label=temperatureLabel></line-chart>
      <line-chart :data="heartrateData" :labels="labels" :label=heartrateLabel></line-chart>
    </div>
  </div>
</template>

<script>

import LineChart from './line-chart.js'

var loginURL = 'http://192.168.92.133:9000/healthmonitorapi/auth/login'
var registerURL = 'http://192.168.92.133:9000/healthmonitorapi/auth/register'
var deviceDataURL = 'http://192.168.92.133:9000/healthmonitorapi/entities/devices/data';
var deviceInfoURL = 'http://192.168.92.133:9000/healthmonitorapi/entities/devices/info'

export default {
  name: 'LoginPage',
  components: { LineChart },
  data: function() {
    return {
      registerUsername: "",
      registerPassword: "",

      loginUsername: "",
      loginPassword: "",

      loginOK: false,

      username: "",
      password: "",

      deviceInfo: null,

      timer: null,
      temperatureLabel: "temperature",
      heartrateLabel: "heartrate",
      temperatureData: [],
      heartrateData: [],
      labels: [],
    }
  },
  methods:{
    registerUser: function() {
      this.$http.post(registerURL, {
        username: this.registerUsername,
        password: this.registerPassword,
      }).then(function(response) {
        this.registerUsername = ""
        this.registerPassword = ""
        
        if (response.statusText == "OK") {
          alert("Register OK!")
        }
      }, function(error) {
        this.registerUsername = ""
        this.registerPassword = ""

        alert("Register failed!")
        console.log(error)
      });
    },
    loginUser: function() {
      console.log(this.loginUsername, this.loginPassword)
      this.$http.post(loginURL, {
        username: this.loginUsername,
        password: this.loginPassword,
      }).then(function(response) {
        if (response.statusText == "OK") {
          this.loginOK = true
          this.username = this.loginUsername
          this.password = this.loginPassword
          this.loginUsername = ""
          this.loginPassword = ""
          this.initDeviceData()
          alert("Login OK!")
        }
      }, function(error) {
        this.loginUsername = ""
        this.loginPassword = ""

        if (error.statusText == "Forbidden") {
          alert("Login failed due to incorrect credetials!")
        } else {
          alert("Login failed due to server error!")
        }
        console.log(error)
      });
    },
    getDeviceInfo: function(){
      this.$http.get(deviceInfoURL, {params: {username: this.username, password: this.password, did: "device-test"}}).then(function(response){
        if (response.statusText == "OK") {
          this.deviceInfo = response.data
        }
      }, function(error){
        this.loginOK = false;
        if (error.statusText == "Forbidden") {
          alert("Login failed!")
        }
      });
    },
    getDeviceData: function(){
      this.$http.get(deviceDataURL, {params: {username: this.username, password: this.password, did: "device-test"}}).then(function(response){
        if (response.statusText == "OK") {
          this.labels = response.data.data.map(datapoint => datapoint.timestamp)
          this.temperatureData = response.data.data.map(datapoint => datapoint.temperature)
          this.heartrateData = response.data.data.map(datapoint => datapoint.heart_rate)
          console.log(this.labels)
          console.log(this.temperatureData)
          console.log(this.heartrateData)
        }
      }, function(error){
        this.loginOK = false;
        if (error.statusText == "Forbidden") {
          alert("Login failed!")
        }
      });
    },
    refreshDeviceData: function() {
      this.getDeviceInfo()
      this.getDeviceData()
    },
  },
  mounted: function () {
    setInterval(this.refreshDeviceData, 5000)
  },
  beforeDestroy: function() {
    clearInterval(this.timer)
  }
}
</script>

<style scoped>
h3 {
  margin: 40px 0 0;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
</style>

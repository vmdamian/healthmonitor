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
      <h2>Selected time interval {{ selectedInterval }}</h2>
      <select v-model="selectedInterval">
        <option v-for="(interval, value) in possibleIntervals" :key="value">
          {{interval}}
        </option>
      </select>
      <h2>Selected device {{ selectedDevice }}</h2>
      <select v-model="selectedDevice">
        <option v-for="device in possibleDevices" :key="device">
          {{device}}
        </option>
      </select>
      <h2>Requested device info</h2>
      <span> {{ deviceInfo }} </span>
      <h2>Device data plotted below</h2>
      <line-chart :data="temperatureData" :labels="labels" :label=temperatureLabel></line-chart>
      <line-chart :data="heartrateData" :labels="labels" :label=heartrateLabel></line-chart>
    </div>
  </div>
</template>

<script>
  import LineChart from './LineChart.vue'

  const baseURL = 'http://192.168.92.133:9000'
  const loginPath = '/healthmonitorapi/auth/login'
  const registerPath = '/healthmonitorapi/auth/register'
  const deviceDataPath = '/healthmonitorapi/entities/devices/data'
  const deviceInfoPath = '/healthmonitorapi/entities/devices/info'
  const userDevicesPath = '/healthmonitorapi/entities/users/devices'

  export default {
  name: 'LoginPage',
  components: { LineChart },
  data: function() {
    return {
      registerUsername: "",
      registerPassword: "",

      possibleIntervals: {
        LAST_MINUTE: 1,
        LAST_FIVE_MINUTE: 5,
        LAST_FIFTEEN_MINUTES: 15
      },

      selectedInterval: 1,

      possibleDevices: [
      ],

      selectedDevice: '',

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
      this.$http.post(baseURL + registerPath, {
        username: this.registerUsername,
        password: this.registerPassword,
      }).then(function(response) {
        this.registerUsername = ""
        this.registerPassword = ""
        
        if (response.statusText === "OK") {
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
      this.$http.post(baseURL + loginPath, {
        username: this.loginUsername,
        password: this.loginPassword,
      }).then(function(response) {
        if (response.statusText === "OK") {
          this.token = response.data.token
          if (this.token === "") {
            return
          }
          this.loginOK = true
          this.getDevicesForUser()
          this.timer = setInterval(this.refreshDeviceData, 5000)
          this.username = this.loginUsername
          this.password = this.loginPassword
          this.loginUsername = ""
          this.loginPassword = ""
          alert("Login OK!")
        }
      }, function(error) {
        this.loginUsername = ""
        this.loginPassword = ""

        if (error.statusText === "Forbidden") {
          alert("Login failed due to incorrect credentials!")
        } else {
          alert("Login failed due to server error!")
        }
      });
    },
    getDevicesForUser: function() {
      this.$http.get(baseURL + userDevicesPath, {headers: {Authorization: 'Bearer ' + this.token}}).then(function(response){
        if (response.statusText === "OK") {
          this.possibleDevices = response.data.user_devices
          if (this.possibleDevices != null) {
            if (this.possibleDevices.length > 0) {
              this.selectedDevice = this.possibleDevices[0]
            }
          }
        }
      }, function(error){
        this.loginOK = false;
        clearInterval(this.timer)
        if (error.statusText === "Forbidden") {
          alert("Login failed!")
        }
      });
    },
    getDeviceInfo: function(){
      if (this.selectedDevice === '') {
        return
      }

      this.$http.get(baseURL + deviceInfoPath, {params: {did: this.selectedDevice}, headers: {Authorization: 'Bearer ' + this.token}}).then(function(response){
        if (response.statusText === "OK") {
          this.deviceInfo = response.data
        }
      }, function(error){
        this.loginOK = false;
        clearInterval(this.timer)
        if (error.statusText === "Forbidden") {
          alert("Login failed!")
        }
      });
    },
    getDeviceData: function(since){
      if (this.selectedDevice === '') {
        return
      }

      this.$http.get(baseURL + deviceDataPath, {params: { did: this.selectedDevice, since: since}, headers: {Authorization: 'Bearer ' + this.token}}).then(function(response){
        if (response.statusText === "OK") {
          this.labels = response.data.data.map(datapoint => datapoint.timestamp)
          this.temperatureData = response.data.data.map(datapoint => datapoint.temperature)
          this.heartrateData = response.data.data.map(datapoint => datapoint.heart_rate)
        }
      }, function(error){
        this.loginOK = false;
        clearInterval(this.timer)
        if (error.statusText === "Forbidden") {
          alert("Login failed!")
        }
      });
    },
    refreshDeviceData: function() {
      const nowTimestamp = Math.round(+new Date() / 1000)
      const minuteAgoTimestamp = nowTimestamp - 60 * this.selectedInterval

      this.getDeviceInfo()
      this.getDeviceData(minuteAgoTimestamp)
    },
  },
  mounted: function () {},
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

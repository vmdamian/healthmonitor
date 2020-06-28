<template>
  <div class="hello">

    <h1>HealthMonitor Welcome Page</h1>

    <div v-if="!loginOK">

      <h2>Complete the following form to register a new user</h2>
      <input v-model="registerUsername" placeholder="username">
      <input v-model="registerPassword" placeholder="password" type="password">
      <input v-model="registerPhone" placeholder="phoneNumber">
      <button v-on:click="registerUser">Register</button>
      <h2>Complete the following form to login with your existing user</h2>
      <input v-model="loginUsername" placeholder="username">
      <input v-model="loginPassword" placeholder="password" type="password">
      <button v-on:click="loginUser">Login</button>

    </div>

    <div v-if="loginOK">

      <div class="container_selectors" v-if="loginOK">
        <div class="col">
          <h2>Selected time interval {{ selectedInterval }} (minutes)</h2>
          <select v-model="selectedInterval">
            <option v-for="(interval, value) in possibleIntervals" :key="value">
              {{interval}}
            </option>
          </select>
        </div>


        <div class="col">
          <h2>Selected refresh frequency {{ selectedFrequency }} (seconds) </h2>
          <select v-model="selectedFrequency" @change="onFrequencyChange()">
            <option v-for="(frequency, value) in possibleFrequencies" :key="value">
              {{frequency}}
            </option>
          </select>
        </div>
      </div>

      <div class="container_selectors">
        <div class="col">
          <h2>Selected device {{ selectedDevice }}</h2>
          <select v-model="selectedDevice">
            <option v-for="device in possibleDevices" :key="device">
              {{device}}
            </option>
          </select>
        </div>
        <div class="col">
          <h2>Device ID</h2>
          <input v-model="operationDevice" placeholder="deviceID">
          <h2>Device actions</h2>
          <button v-on:click="onDeviceAdd">Add</button>
          <button v-on:click="onDeviceRemove">Delete</button>
          <button v-on:click="onDeviceSubscribe">Subscribe</button>
          <button v-on:click="onDeviceUnsubscribe">Unsubscribe</button>
        </div>
      </div>


      <div v-if="deviceInfo !== null" class="container_data">
        <h2>Requested device info</h2>
        <h5>Device ID: {{deviceInfo.device_info.did}}</h5>
        <h5>Last seen timestamp: {{deviceInfo.device_info.last_seen_timestamp}}</h5>
        <h5>Last validation timestamp: {{deviceInfo.device_info.last_validation_timestamp}}</h5>
        <h5>Patient Name: {{deviceInfo.device_info.patient_name}}</h5>
      </div>


      <div v-if="deviceInfo !== null" class="container_data">
      <h3>Temperature</h3>
      <line-chart :data="temperatureData" :labels="labels" :label=temperatureLabel></line-chart>
      <h2 v-if="temperatureAlerts.length > 0">Alerts</h2>
      <div v-if="temperatureAlerts.length > 0" class="container_alerts">
        <div v-for="(alert, index) in temperatureAlerts" :key="index" v-bind:class="alert.status ==='ACTIVE' ? 'alert-active' : 'alert-resolved'">
          <h5>Alert Type: {{alert.alert_type}}</h5>
          <h5>Created Timestamp: {{alert.created_timestamp}}</h5>
          <h5>Last Active Timestamp: {{alert.last_active_timestamp}}</h5>
          <h5 v-if="alert.status === 'RESOLVED'">Resolved Timestamp: {{alert.resolved_timestamp}}</h5>
          <h5>Status: {{alert.status}}</h5>
        </div>
      </div>

      </div>


      <div v-if="deviceInfo !== null" class="container_data">
      <h3>Heart rate</h3>
      <line-chart :data="heartrateData" :labels="labels" :label=heartrateLabel></line-chart>
        <h2 v-if="pulseAlerts.length > 0">Alerts</h2>
        <div v-if="pulseAlerts.length > 0" class="container_alerts">
          <div v-for="(alert, index) in pulseAlerts" :key="index" v-bind:class="alert.status ==='ACTIVE' ? 'alert-active' : 'alert-resolved'">
            <h5>Alert Type: {{alert.alert_type}}</h5>
            <h5>Created Timestamp: {{alert.created_timestamp}}</h5>
            <h5>Last Active Timestamp: {{alert.last_active_timestamp}}</h5>
            <h5 v-if="alert.status === 'RESOLVED'">Resolved Timestamp: {{alert.resolved_timestamp}}</h5>
            <h5>Status: {{alert.status}}</h5>
          </div>
        </div>
      </div>


      <div v-if="deviceInfo !== null" class="container_data">
      <h3>ECG</h3>
      <line-chart :data="ecgData" :labels="labels" :label=ecgLabel></line-chart>
        <h2 v-if="ecgAlerts.length > 0">Alerts</h2>
        <div v-if="ecgAlerts.length > 0" class="container_alerts">
          <div v-for="(alert, index) in ecgAlerts" :key="index" v-bind:class="alert.status ==='ACTIVE' ? 'alert-active' : 'alert-resolved'">
            <h5>Alert Type: {{alert.alert_type}}</h5>
            <h5>Created Timestamp: {{alert.created_timestamp}}</h5>
            <h5>Last Active Timestamp: {{alert.last_active_timestamp}}</h5>
            <h5 v-if="alert.status === 'RESOLVED'">Resolved Timestamp: {{alert.resolved_timestamp}}</h5>
            <h5>Status: {{alert.status}}</h5>
          </div>
        </div>
      </div>


      <div v-if="deviceInfo !== null" class="container_data">
      <h3>Blood Oxygen Saturation</h3>
      <line-chart :data="oxygenData" :labels="labels" :label=oxygenLabel></line-chart>
        <h2 v-if="oxygenAlerts.length > 0">Alerts</h2>
        <div v-if="oxygenAlerts.length > 0" class="container_alerts">
          <div v-for="(alert, index) in oxygenAlerts" :key="index" v-bind:class="alert.status ==='ACTIVE' ? 'alert-active' : 'alert-resolved'">
            <h5>Alert Type: {{alert.alert_type}}</h5>
            <h5>Created Timestamp: {{alert.created_timestamp}}</h5>
            <h5>Last Active Timestamp: {{alert.last_active_timestamp}}</h5>
            <h5 v-if="alert.status === 'RESOLVED'">Resolved Timestamp: {{alert.resolved_timestamp}}</h5>
            <h5>Status: {{alert.status}}</h5>
          </div>
        </div>
    </div>
    </div>
  </div>

</template>

<script>
  import LineChart from './LineChart.vue'

  const baseURL = 'http://healthmonitor-d2400c9ab166d3ea.elb.us-east-2.amazonaws.com'
  const loginPath = '/healthmonitorapi/auth/login'
  const registerPath = '/healthmonitorapi/auth/register'
  const deviceDataPath = '/healthmonitorapi/entities/devices/data'
  const deviceInfoPath = '/healthmonitorapi/entities/devices/info'
  const deviceAlertsPath = '/healthmonitorapi/entities/devices/alerts'
  const userDevicesPath = '/healthmonitorapi/entities/users/devices'
  const userSubscriptionsPath = '/healthmonitorapi/entities/users/subscriptions'

  export default {
  name: 'LoginPage',
  components: { LineChart },
  data: function() {
    return {
      registerUsername: "",
      registerPassword: "",
      registerPhone: "",
      operationDevice: "",

      possibleIntervals: {
        LAST_MINUTE: 1,
        LAST_FIVE_MINUTES: 5,
        LAST_FIFTEEN_MINUTES: 15,
        LAST_THIRTY_MINUTES: 30,
        LAST_SIXTY_MINUTES: 60,
      },

      possibleFrequencies: {
        EVERY_FIVE_SECONDS: 5,
        EVERY_FIFTEEN_SECONDS: 15,
        EVERY_THIRTY_SECONDS: 30,
        EVERY_MINUTE: 60,
        EVERY_FIVE_MINUTES: 300,
      },

      selectedInterval: 1,
      selectedFrequency: 5,

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
      temperatureLabel: "TEMPERATURE",
      temperatureData: [],
      heartrateLabel: "HEART RATE",
      heartrateData: [],
      oxygenLabel: "SPO2",
      oxygenData: [],
      ecgLabel: "ECG",
      ecgData: [],
      labels: [],
      deviceAlerts: [],
      temperatureAlerts: [],
      oxygenAlerts: [],
      ecgAlerts: [],
      pulseAlerts: [],
    }
  },
  methods:{
    onFrequencyChange: function() {
      clearInterval(this.timer)
      this.timer = setInterval(this.refreshDeviceData, this.selectedFrequency * 1000)
    },
    registerUser: function() {
      this.$http.post(baseURL + registerPath, {
        username: this.registerUsername,
        password: this.registerPassword,
        phone_number: this.registerPhone
      }).then(function(response) {
        this.registerUsername = ""
        this.registerPassword = ""
        this.registerPhone = ""
        
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
          this.timer = setInterval(this.refreshDeviceData, this.selectedFrequency * 1000)
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
          this.labels = response.data.device_dataset.data.map(function(datapoint) {
            const date = new Date(datapoint.timestamp)
            const seconds = date.getSeconds()
            const minutes = date.getMinutes()
            const hour = date.getHours()
            return hour + ":" + minutes + ":" + seconds
          })
          this.temperatureData = response.data.device_dataset.data.map(datapoint => datapoint.temperature)
          this.heartrateData = response.data.device_dataset.data.map(datapoint => datapoint.heart_rate)
          this.ecgData = response.data.device_dataset.data.map(datapoint => datapoint.heart_ecg)
          this.oxygenData = response.data.device_dataset.data.map(datapoint => datapoint.spo2)
        }
      }, function(error){
        this.loginOK = false;
        clearInterval(this.timer)
        if (error.statusText === "Forbidden") {
          alert("Login failed!")
        }
      });
    },
    getDeviceAlerts: function(){
      if (this.selectedDevice === '') {
        return
      }
      this.$http.get(baseURL + deviceAlertsPath, {params: {did: this.selectedDevice}, headers: {Authorization: 'Bearer ' + this.token}}).then(function(response){
        if (response.statusText === "OK") {
          this.deviceAlerts = response.data.alerts
          this.temperatureAlerts = this.deviceAlerts.filter(function(alert) {
            return alert.alert_type === "TEMPERATURE_HIGH" || alert.alert_type === "TEMPERATURE_LOW"
          })
          this.ecgAlerts = this.deviceAlerts.filter(function(alert) {
            return alert.alert_type === "ECG_HIGH" || alert.alert_type === "ECG_LOW"
          })
          this.oxygenAlerts = this.deviceAlerts.filter(function(alert) {
            return alert.alert_type === "SP02_HIGH" || alert.alert_type === "SP02_LOW"
          })
          this.pulseAlerts = this.deviceAlerts.filter(function(alert) {
            return alert.alert_type === "HEARTRATE_HIGH" || alert.alert_type === "HEARTRATE_LOW"
          })
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
      this.getDeviceAlerts()
    },
    onDeviceAdd: function() {
      if (this.operationDevice === '') {
        return
      }
      this.$http.post(baseURL + userDevicesPath, {
        user_device: this.operationDevice,
      }, {headers: {Authorization: 'Bearer ' + this.token}}).then(function(response) {
        if (response.statusText === "OK") {
          this.getDevicesForUser()
          alert("OK!")
        }
      }, function(error) {
        if (error.statusText === "Forbidden") {
          alert("Operation failed due to incorrect credentials!")
        } else {
          alert("Operation failed due to server error!")
        }
      });
    },
    onDeviceRemove: function() {
      if (this.operationDevice === '') {
        return
      }
      this.$http.delete(baseURL + userDevicesPath, {params: { did: this.selectedDevice},headers: {Authorization: 'Bearer ' + this.token}}).then(function(response) {
        if (response.statusText === "OK") {
          this.getDevicesForUser()
          alert("OK!")
        }
      }, function(error) {
        if (error.statusText === "Forbidden") {
          alert("Operation failed due to incorrect credentials!")
        } else {
          alert("Operation failed due to server error!")
        }
      });
    },
    onDeviceSubscribe: function() {
      if (this.operationDevice === '') {
        return
      }
      this.$http.post(baseURL + userSubscriptionsPath, {
        did: this.operationDevice,
      }, {headers: {Authorization: 'Bearer ' + this.token}}).then(function(response) {
        if (response.statusText === "OK") {
          this.getDevicesForUser()
          alert("OK!")
        }
      }, function(error) {
        if (error.statusText === "Forbidden") {
          alert("Operation failed due to incorrect credentials!")
        } else {
          console.log(error)
          console.log(error.statusText)
          alert("Operation failed due to server error!")
        }
      });
    },
    onDeviceUnsubscribe: function() {
      if (this.operationDevice === '') {
        return
      }
      this.$http.delete(baseURL + userSubscriptionsPath, {params: { did: this.selectedDevice},headers: {Authorization: 'Bearer ' + this.token}}).then(function(response) {
        if (response.statusText === "OK") {
          this.getDevicesForUser()
          alert("OK!")
        }
      }, function(error) {
        if (error.statusText === "Forbidden") {
          alert("Operation failed due to incorrect credentials!")
        } else {
          alert("Operation failed due to server error!")
        }
      });
    },
  },
  mounted: function () {},
  beforeDestroy: function() {
    clearInterval(this.timer)
  }
}
</script>

<style scoped>
  .hello {
    background: lightblue;
  }
  .container_selectors {
    margin: 20px;
    border: 1px solid;
    display: flex;
    background: white;
  }

  .alert-active {
    border: 1px solid;
    margin: 20px;
    background: red;
  }

  .alert-resolved {
    border: 1px solid;
    margin: 20px;
    background: yellow;
  }

  .container_data{
    margin: 20px;
    border: 1px solid;
    background: white;
  }
  .col {
    margin: 10px;
    border: 1px solid;
    flex: 1;
  }
  .container_alerts {
    border: 1px solid;
    overflow: auto;
    white-space: nowrap;
    display: flex;
  }

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

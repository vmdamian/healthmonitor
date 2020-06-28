<template>
  <div id="app">
    <LoginPage/>
  </div>
</template>

<script>
import LoginPage from './components/LoginPage.vue'

export default {
  name: 'HealthMonitorUI',
  components: {
    LoginPage
  }
}
</script>

<style>
  html {
    background: lightblue;
  }
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  background: lightblue;
}
</style>

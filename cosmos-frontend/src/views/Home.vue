<template>
  <div class="home">
    <v-row style="height: 100%" no-gutters justify="center" align="center">
      <v-col class="text-center" cols="8">
        <div class="text-h2 font-weight-thin mb-8 white--text">
          COSMOS
        </div>
      </v-col>
    </v-row>
  </div>
</template>

<script>
export default {
  name: 'Home'
}
</script>

<style scoped>
/* Accessing static files in assets folder - https://stackoverflow.com/a/52097176 */
.home {
  background-image: url('~@/assets/material-design-gray-and-blue.jpg');
  background-position: center;
  background-repeat: no-repeat;
  background-size: cover;
  height: 100%;
}
</style>

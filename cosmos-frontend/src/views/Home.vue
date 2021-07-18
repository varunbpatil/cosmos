<template>
  <div class="home">
    <v-row style="height: 100%" no-gutters justify="center" align="center">
      <v-col class="text-center" cols="8">
        <div class="text-h2 font-weight-thin mb-8 white--text">
          COSMOS
        </div>
        <div class="text-h6 white--text font-italic font-weight-light">
          “<span class="font-weight-medium">Cosmos</span> is a Greek word for the order of the universe.
          It is, in a way, the opposite of <span class="font-weight-medium">Chaos</span>.
          It implies the deep <span class="font-weight-medium">interconnectedness</span> of all things.
          It conveys awe for the intricate and subtle way in which the universe is put together.”
        </div>
        <div class="text-h6 white--text font-italic font-weight-light text-right">
          - Carl Sagan
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

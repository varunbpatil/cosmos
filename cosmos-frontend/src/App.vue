<template>
  <v-app>
    <Navbar />

    <v-main class="grey lighten-2">
      <router-view />
    </v-main>

  </v-app>
</template>

<script>
import Navbar from '@/components/Navbar'

export default {
  name: 'App',

  components: {
    Navbar
  },

  data: () => ({
    //
  }),
};
</script>

<style>
.text-subtitle-1 {
  line-height: 1.5rem !important;
}
</style>

<template>
  <v-container>

    <v-tabs fixed-tabs show-arrows background-color="transparent" color="indigo darken-1">
      <v-tab to="/endpoints/sources">sources</v-tab>
      <v-tab to="/endpoints/destinations">destinations</v-tab>
    </v-tabs>

    <!--
    Since the same component - "EndpointsByType" - is being used for both
    /endpoints/sources and /endpoints/destinations, we need to tell Vue
    to reload the component when switching between the tabs. This is done
    using the "key" to router-view.
    -->
    <router-view :key="$route.fullPath"></router-view>

  </v-container>
</template>

<script>
export default {

}
</script>

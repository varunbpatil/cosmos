<template>
  <div>
    <!-- app bar -->
    <v-app-bar dense app color="grey lighten-2">
      <v-app-bar-nav-icon @click="drawer = !drawer"></v-app-bar-nav-icon>
      <router-link to="/" class="text-decoration-none">
        <v-app-bar-title>
          <div class="px-2 text-h6 grey--text text--darken-2">COSMOS</div>
        </v-app-bar-title>
      </router-link>
    </v-app-bar>

    <!-- navigation drawer -->
    <v-navigation-drawer
      app
      dark
      v-model="drawer"
      color="indigo darken-1"
      :expand-on-hover="$vuetify.breakpoint.lgAndUp"
    >
      <!--
      Using "flex-column" to lay out the elements of the navigation-drawer vertically.
      In this mode, justify-* doesn't do anything because the height of the page is not fixed
      unlike in "flex-row"(default) where the width of the page is fixed.
      -->
      <v-row class="flex-column" no-gutters>

        <!-- avatar -->
        <!-- "align-self-center" horizontally centers the avatar when "flex-column" is set on the parent -->
        <router-link to="/" class="mt-2 align-self-center text-decoration-none">
          <v-avatar color="indigo">
            <v-icon large>mdi-account</v-icon>
          </v-avatar>
        </router-link>

        <!-- links -->
        <v-list class="mt-16">
          <v-list-item v-for="link in links" :key="link.name" :to="link.path">
            <v-list-item-icon>
              <v-icon>{{ link.icon }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content>
              <v-list-item-title class="font-weight-medium">{{ link.name }}</v-list-item-title>
            </v-list-item-content>
          </v-list-item>
        </v-list>

      </v-row>
    </v-navigation-drawer>
  </div>
</template>

<script>
export default {
  data() {
    return {
      drawer: true,
      links: [
        {name: "Syncs", icon: "mdi-sync", path: "/syncs"},
        {name: "Endpoints", icon: "mdi-target", path: "/endpoints"},
        {name: "Connectors", icon: "mdi-handshake", path: "/connectors"}
      ]
    }
  }
}
</script>

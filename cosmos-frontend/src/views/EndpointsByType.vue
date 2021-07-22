<template>
  <v-container class="mt-4">

    <!--snackbar-->
    <v-snackbar v-model="snackbarToggle" :timeout="10000">
      <span class="font-weight-medium">{{ snackbarText }}</span>
      <template v-slot:action="{ attrs }">
        <v-btn color="yellow" text v-bind="attrs" @click="snackbarToggle = !snackbarToggle">CLOSE</v-btn>
      </template>
    </v-snackbar>

    <!-- create-new-endpoint dialog -->
    <CreateEndpoint :endpointType="endpointType" @create="snackbar('created', ...arguments)"></CreateEndpoint>

    <!-- list of endpoints -->
    <v-card flat v-for="e in endpoints" :key="e.id" class="mt-4">
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="4" lg="4" class="py-1">
            <div class="font-weight-medium indigo--text">Name</div>
            <div class="text-subtitle-1">{{ e.name }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="4" lg="3" class="py-1">
            <div class="font-weight-medium indigo--text">Connector</div>
            <div class="text-subtitle-1">{{ e.connector.name }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="4" lg="3" class="py-1">
            <div v-if="endpointType === 'source'" class="font-weight-medium indigo--text">Last discovered</div>
            <div v-if="endpointType === 'source'" class="text-subtitle-1">{{ lastDiscovered(e) }}</div>
          </v-col>
          <v-col cols="12" sm="3" md="2" lg="2" align-self="center" class="py-1">
            <v-row no-gutters align="center">
              <!--rediscover-->
              <v-col>
                <v-tooltip bottom>
                  <template v-slot:activator="{ on, attrs }">
                    <v-btn
                      icon
                      large
                      v-bind="attrs"
                      v-on="on"
                      v-if="endpointType === 'source'"
                      color="indigo"
                      @click="rediscover(e)"
                    >
                      <v-icon>mdi-refresh</v-icon>
                    </v-btn>
                  </template>
                  <span class="font-weight-medium">Rediscover</span>
                </v-tooltip>
              </v-col>

              <!-- edit-endpoint dialog -->
              <v-col>
                <EditEndpoint
                  :endpoint="e"
                  @delete="snackbar('deleted', ...arguments)"
                  @save="snackbar('saved', ...arguments)"
                ></EditEndpoint>
              </v-col>
            </v-row>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

  </v-container>
</template>

<script>
import EditEndpoint from '@/components/EditEndpoint'
import CreateEndpoint from '@/components/CreateEndpoint'
const _ = require('lodash')

export default {
  components: {
    EditEndpoint,
    CreateEndpoint
  },

  data() {
    return {
      endpoints: null,
      totalEndpoints: null,
      snackbarToggle: false,
      snackbarText: null,
      intervalID: null,
    }
  },

  computed: {
    endpointType() {
      switch(this.$route.params.type) {
        case "sources":
          return "source"
        case "destinations":
          return "destination"
        default:
          return null
      }
    }
  },

  methods: {
    fetchEndpointsByType(type) {
      this.$axios
        .get("api/v1/endpoints?type=" + type)
        .then(response => {
          this.endpoints = response.data.endpoints
          this.totalEndpoints = response.data.totalEndpoints
        })
    },

    // See https://stackoverflow.com/questions/53738919/emit-event-with-parameters-in-vue
    snackbar(action, target) {
      // Remove the previous snackbar text (if any).
      this.snackbarToggle = false
      // Wait until the snackbar is removed from the DOM before rendering the new snackbar text.
      // See https://vuejsdevelopers.com/2019/01/22/vue-what-is-next-tick/
      this.$nextTick(() => {
        this.snackbarText = "Successfully " + action + " " + target + " endpoint"
        this.snackbarToggle = true
      })
    },

    rediscover(endpoint) {
      this.snackbar("triggered rediscovery on", endpoint.name)
      this.$axios
        .post(`/api/v1/endpoints/${endpoint.id}/rediscover`)
        .then(() => {
          this.snackbar("rediscovered", endpoint.name)
        })
    },

    lastDiscovered(endpoint) {
      let milliseconds = new Date() - Date.parse(endpoint.lastDiscovered)

      if (milliseconds < 60000) { // 1 minute
        return "a few seconds ago"
      } else if (milliseconds < 600000) { // 10 minutes
        return "a few minutes ago"
      } else {
        var d = Math.floor(milliseconds / 86400000)
        var h = Math.floor(milliseconds % 86400000 / 3600000)
        var m = Math.floor(milliseconds % 86400000 % 3600000 / 60000)

        var dDisplay = d > 0 ? d + "d " : "";
        var hDisplay = h > 0 ? h + "h " : "";
        var mDisplay = m > 0 ? m + "m" : "";
        return dDisplay + hDisplay + mDisplay + " ago";
      }
    }
  },

  mounted() {
    // If the endpoint type is not expected, redirect to /endpoints.
    if (!this.endpointType) {
      this.$router.push("/endpoints")
    }

    // Create a throttled version of the fetchEndpointsByType function
    // which executes at most once every 1000ms.
    let fn = _.throttle(this.fetchEndpointsByType, 1000)

    // First time endpoint fetch.
    fn(this.endpointType)

    // Supabase realtime updates.
    this.$connectorChanges.on("*", () => fn(this.endpointType))
    this.$endpointChanges.on("*", () => fn(this.endpointType))

    // Do a complete refresh every 5000ms.
    var v = this // Cannot access "this" inside setInterval.
    this.intervalID = setInterval(function() { fn(v.endpointType) }, 5000)
  },

  beforeDestroy() {
    clearInterval(this.intervalID)
    this.$connectorChanges.off("*")
    this.$endpointChanges.off("*")
  }
}
</script>

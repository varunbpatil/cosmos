<template>
  <v-container class="mt-4">
    <!--snackbar-->
    <v-snackbar v-model="snackbarToggle" :timeout="10000">
      <span class="font-weight-medium">{{ snackbarText }}</span>
      <template v-slot:action="{ attrs }">
        <v-btn color="yellow" text v-bind="attrs" @click="snackbarToggle = !snackbarToggle">CLOSE</v-btn>
      </template>
    </v-snackbar>

    <!-- create-new-sync dialog -->
    <CreateSync v-if="$route.name === 'Syncs'" @create="snackbar('Successfully created', ...arguments)"></CreateSync>

    <!-- list of syncs -->
    <v-card flat v-for="s in syncs" :key="s.id" :class="`mt-4 ${getSyncStatus(s)}`" :to="`/syncs/${s.id}`">
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="3" lg="3" class="py-1">
            <div class="font-weight-medium indigo--text">Name</div>
            <div class="text-subtitle-1">{{ s.name }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" class="py-1">
            <div class="font-weight-medium indigo--text">Source endpoint</div>
            <div class="text-subtitle-1">{{ s.sourceEndpoint.name }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="2" class="py-1">
            <div class="font-weight-medium indigo--text">Destination endpoint</div>
            <div class="text-subtitle-1">{{ s.destinationEndpoint.name }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="3" lg="3" class="py-1">
            <div class="font-weight-medium indigo--text">Last successful run</div>
            <div class="text-subtitle-1">{{ lastSuccessfulRun(s) }}</div>
          </v-col>
          <v-col cols="12" sm="4" md="3" lg="2" align-self="center" class="py-1">
            <v-row no-gutters align="center">
              <!--toggle-->
              <v-col>
                <v-tooltip bottom>
                  <template v-slot:activator="{ on, attrs }">
                    <div v-bind="attrs" v-on="on">
                      <v-switch
                        inset
                        hide-details
                        class="ma-0 pa-0"
                        color="indigo"
                        v-model="s.enabled"
                        @change="toggleSync(s.id, ...arguments)"
                        @click.stop.prevent
                      >
                      </v-switch>
                    </div>
                  </template>
                  <span class="font-weight-medium">{{ s.enabled ? "Disable" : "Enable" }} Sync</span>
                </v-tooltip>
              </v-col>

              <!--sync now button-->
              <v-col>
                <v-tooltip bottom>
                  <template v-slot:activator="{ on, attrs }">
                    <v-btn icon large v-bind="attrs" v-on="on" color="indigo" class="ma-0 pa-0" @click.stop.prevent="syncNow(s.id, s.name)">
                      <v-icon>mdi-play</v-icon>
                    </v-btn>
                  </template>
                  <span class="font-weight-medium">Sync Now</span>
                </v-tooltip>
              </v-col>

              <!-- edit-sync dialog -->
              <v-col>
                <EditSync
                  :sync="s"
                  @delete="snackbar('Successfully deleted', ...arguments)"
                  @save="snackbar('Successfully saved', ...arguments)"
                  @clearIncrementalState="snackbar('Successfully cleared incremental state for', ...arguments)"
                  @clearDestinationData="snackbar('Launched a new run to clear destination data for', ...arguments)"
                ></EditSync>
              </v-col>
            </v-row>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <router-view></router-view>

  </v-container>
</template>

<script>
import CreateSync from '@/components/CreateSync'
import EditSync from '@/components/EditSync'
const _ = require('lodash')

export default {
  components: {
    CreateSync,
    EditSync
  },

  data() {
    return {
      syncs: null,
      totalSyncs: null,
      snackbarToggle: false,
      snackbarText: null,
      intervalID: null,
    }
  },

  watch: {
    '$route.name': function(val) {
      if (val !== "Syncs") {
        // User has clicked on a particular sync. Filter out all others.
        this.syncs = this.syncs.filter(obj => obj.id == this.$route.params.syncID)
        this.totalSyncs = 1
      } else {
        this.fetchSyncs()
      }
    }
  },

  methods: {
    fetchSyncs() {
      if (this.$route.name !== 'Syncs') {
        // Fetch only that particular sync.
        this.$axios
          .get(`/api/v1/syncs/${this.$route.params.syncID}`)
          .then(response => {
            this.syncs = response.data.syncs
            this.totalSyncs = response.data.totalSyncs
          })
      } else {
        // Fetch all syncs.
        this.$axios
          .get("/api/v1/syncs")
          .then(response => {
            this.syncs = response.data.syncs
            this.totalSyncs = response.data.totalSyncs
          })
      }
    },

    // See https://stackoverflow.com/questions/53738919/emit-event-with-parameters-in-vue
    snackbar(action, target, error=null) {
      // Remove the previous snackbar text (if any).
      this.snackbarToggle = false
      // Wait until the snackbar is removed from the DOM before rendering the new snackbar text.
      // See https://vuejsdevelopers.com/2019/01/22/vue-what-is-next-tick/
      this.$nextTick(() => {
        this.snackbarText = action + " " + target + " sync" + (error !== null ? ". " + error : "")
        this.snackbarToggle = true
      })
    },

    toggleSync(id, val) {
      this.$axios.patch(`/api/v1/syncs/${id}`, {enabled: val})
    },

    getSyncStatus(sync) {
      if (sync.lastRun) {
        return 'sync-' + sync.lastRun.status.toLowerCase()
      } else {
        return "sync-unknown"
      }
    },

    syncNow(id, name) {
      this.$axios
        .post(`/api/v1/syncs/${id}/sync-now`, {})
        .then(() => {
          this.snackbar("Successfully triggered", name)
        })
        .catch((error) => {
          if (error.response) {
            this.snackbar("Failed to trigger", name, error.response.data.error)
          }
        })
    },

    lastSuccessfulRun(sync) {
      if (sync.lastSuccessfulRun) {
        let milliseconds = new Date() - Date.parse(sync.lastSuccessfulRun.executionDate)

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
      } else {
        return "never"
      }
    }
  },

  mounted() {
    // Create a throttled version of the fetchSyncs function
    // which executes at most once every 1000ms.
    let fn = _.throttle(this.fetchSyncs, 1000)

    // First time sync fetch.
    fn()

    // Supabase realtime updates.
    this.$endpointChanges.on("*", () => fn())
    this.$runChanges.on("*", () => fn())
    this.$syncChanges.on("*", () => fn())

    // Do a complete refresh every 5000ms.
    this.intervalID = setInterval(function() { fn() }, 5000)
  },

  beforeDestroy() {
    clearInterval(this.intervalID)
    // This is the opposite of what this.$*Changes.on() would do.
    // There is an off() method, but that removes all callbacks associated with "*".
    // With nested views like the Syncs page, we only want to remove the callbacks we added.
    this.$endpointChanges.bindings.pop()
    this.$runChanges.bindings.pop()
    this.$syncChanges.bindings.pop()
  }
}
</script>

<style scoped>
.sync-queued {
  border-left: 5px solid #eed202;
}
.sync-running {
  border-left: 5px solid #7cfc00;
}
.sync-success {
  border-left: 5px solid #228b22;
}
.sync-failed {
  border-left: 5px solid #ff0000;
}
.sync-canceled {
  border-left: 5px solid #ff0000;
}
.sync-wiped {
  border-left: 5px solid #3333ff;
}
.sync-unknown {
  border-left: 5px solid #888888;
}
</style>

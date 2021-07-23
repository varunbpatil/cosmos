<template>
  <div>
    <ConfirmationDialog ref="confirm" />

    <!--snackbar-->
    <v-snackbar v-model="snackbarToggle" :timeout="10000">
      <span class="font-weight-medium">{{ snackbarText }}</span>
      <template v-slot:action="{ attrs }">
        <v-btn color="yellow" text v-bind="attrs" @click="snackbarToggle = !snackbarToggle">CLOSE</v-btn>
      </template>
    </v-snackbar>

    <!--filter by status or date range-->
    <v-row class="my-4" v-if="$route.name === 'Runs'">
      <v-col cols="12" md="6">
        <v-select
          multiple
          :items="['queued', 'running', 'success', 'failed', 'canceled', 'wiped']"
          v-model="filterStatus"
          label="Filter runs by status"
          :menu-props="{ offsetY: true }"
          prepend-icon="mdi-filter-outline"
          color="indigo"
          item-color="indigo"
          clearable
        ></v-select>
      </v-col>
      <v-col cols="12" md="6">
        <v-menu bottom offset-y :close-on-content-click="false" min-width="auto">
          <template v-slot:activator="{ on, attrs }">
            <v-text-field
              v-bind="attrs"
              v-on="on"
              v-model="dateRangeText"
              label="Filter runs by date range"
              color="indigo"
              readonly
              clearable
              prepend-icon="mdi-calendar-outline"
            ></v-text-field>
          </template>
          <v-date-picker
            range
            no-title
            scrollable
            v-model="dateRange"
            min="1970-01-01"
            :max="new Date().toISOString().substr(0, 10)"
            color="indigo"
          ></v-date-picker>
        </v-menu>
      </v-col>
    </v-row>
    <v-divider v-else class="my-8"></v-divider>

    <div v-if="$route.name === 'Runs' && totalRuns > 0" class="text-subtitle-1 grey--text text--darken-2 mb-4">
      Runs {{ ((page-1)*resultsPerPage)+1 }} - {{ Math.min(totalRuns, ((page-1)*resultsPerPage)+resultsPerPage) }} of {{ totalRuns }}
    </div>

    <v-card flat v-for="r in runs" :key="r.id" :class="`mb-4 ${getRunStatus(r)}`" :to="`/syncs/${$route.params.syncID}/${r.id}`">
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="5" class="py-1">
            <div class="font-weight-medium indigo--text">Execution date</div>
            <div class="text-subtitle-1">{{ formattedDate(r.executionDate) }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="2" class="py-1">
            <div class="font-weight-medium indigo--text">Status</div>
            <div class="text-subtitle-1">{{ r.status }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="2" class="py-1">
            <div class="font-weight-medium indigo--text">Records</div>
            <div class="text-subtitle-1">{{ r.stats.numRecords }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="2" class="py-1">
            <div class="font-weight-medium indigo--text">Time taken</div>
            <div class="text-subtitle-1">{{ timeTaken(r.stats.executionStart, r.stats.executionEnd) }}</div>
          </v-col>
          <v-col v-if="r.status === 'running'" cols="12" sm="6" md="1" align-self="center" class="py-1">
            <v-tooltip bottom>
              <template v-slot:activator="{ on, attrs }">
                <v-btn icon large v-bind="attrs" v-on="on" color="red darken-2" class="ma-0 pa-0" @click.stop.prevent="cancelRun(r.id)">
                  <v-icon>mdi-cancel</v-icon>
                </v-btn>
              </template>
              <span class="font-weight-medium">Cancel Run</span>
            </v-tooltip>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <!--pagination-->
    <div class="text-center mt-8" v-if="totalRuns > resultsPerPage">
      <v-pagination
        v-model="page"
        :length="Math.ceil(totalRuns/resultsPerPage)"
        :total-visible="11"
        color="indigo"
      ></v-pagination>
    </div>

    <router-view></router-view>
  </div>
</template>

<script>
import { format } from 'date-fns'
const _ = require('lodash')

export default {
  components: {
    ConfirmationDialog: () => import("../components/ConfirmationDialog"),
  },

  data() {
    return {
      page: 1,
      resultsPerPage: 10,
      runs: null,
      totalRuns: null,
      intervalID: null,
      filterStatus: null,
      dateRange: [],
      snackbarToggle: false,
      snackbarText: null,
    }
  },

  computed: {
    dateRangeText: {
      get: function() {
        return this.dateRange.join(' ~ ')
      },
      set: function(val) {
        if (val === null) {
          this.dateRange = []
        }
      }
    }
  },

  watch: {
    '$route.name': function(val) {
      if (val !== "Runs") {
        // User has clicked on a particular run. Filter out all others.
        this.runs = this.runs.filter(obj => obj.id == this.$route.params.runID)
        this.totalRuns = 1
      } else {
        this.fetchRuns()
      }
    },

    page: function() {
      this.runs = null
      this.totalRuns = null
      this.fetchRuns()
    },

    filterStatus: function(val) {
      if (val && val.length == 0) {
        this.filterStatus = null
      }
      this.fetchRuns()
    },

    dateRange: function(val) {
      if (val && (val.length == 0 || val.length == 2)) {
        this.fetchRuns()
      }
    }
  },

  methods: {
    formattedDate(date) {
      // Format and display date in local timezone.
      return format(Date.parse(date), "dd MMM yyyy hh:mm:ss a")
    },

    fetchRuns() {
      if (this.$route.name !== 'Runs') {
        // Fetch only that particular run.
        this.$axios
          .get(`api/v1/runs/${this.$route.params.runID}`)
          .then(response => {
            this.runs = response.data.runs
            this.totalRuns = response.data.totalRuns
          })
      } else {
        // Fetch all runs for the given syncID.
        this.$axios
          .post("api/v1/runs", {
            syncID: Number(this.$route.params.syncID),
            status: this.filterStatus,
            dateRange: this.dateRange.length == 2 ? this.sorted(this.dateRange) : null,
            offset: (this.page-1)*this.resultsPerPage,
            limit: this.resultsPerPage
          })
          .then(response => {
            this.runs = response.data.runs
            this.totalRuns = response.data.totalRuns
          })
      }
    },

    sorted(dateRange) {
      let dateRangeCopy = [...dateRange]
      return dateRangeCopy.sort()
    },

    getRunStatus(run) {
      return 'run-' + run.status
    },

    timeTaken(start, end) {
      start = new Date(start)
      end = new Date(end)
      let ms = end.getTime() - start.getTime()
      var h = Math.floor(ms / 3600000)
      var m = Math.floor(ms % 3600000 / 60000)
      var s = Math.floor(ms % 3600000 % 60000 / 1000)

      var hDisplay = h > 0 ? h + "h " : "";
      var mDisplay = m > 0 ? m + "m " : "";
      var sDisplay = s > 0 ? s + "s" : "";
      return hDisplay + mDisplay + sDisplay || "0s";
    },

    snackbar(text) {
      // Remove the previous snackbar text (if any).
      this.snackbarToggle = false
      // Wait until the snackbar is removed from the DOM before rendering the new snackbar text.
      // See https://vuejsdevelopers.com/2019/01/22/vue-what-is-next-tick/
      this.$nextTick(() => {
        this.snackbarText = text
        this.snackbarToggle = true
      })
    },

    async cancelRun(id) {
      if (
          await this.$refs.confirm.open(
            "Confirm", "Are you sure you want to cancel this run?"
          )
      ) {
        this.$axios
          .post(`/api/v1/runs/${id}/cancel`)
          .then(() => {
            this.snackbar("Canceled run")
          })
          .catch((error) => {
            if (error.response) {
              this.snackbar(error.response.data.error)
            }
          })
      }
    }
  },

  mounted() {
    // Create a throttled version of the fetchRuns function
    // which executes at most once every 1000ms.
    let fn = _.throttle(this.fetchRuns, 1000)

    // First time run fetch.
    fn()

    // Supabase realtime updates.
    this.$runChanges.on("*", () => fn())

    // Do a complete refresh every 5000ms.
    this.intervalID = setInterval(function() { fn() }, 5000)
  },

  beforeDestroy() {
    clearInterval(this.intervalID)
    // This is the opposite of what this.$*Changes.on() would do.
    // There is an off() method, but that removes all callbacks associated with "*".
    // With nested views like the Syncs page, we only want to remove the callbacks we added.
    this.$runChanges.bindings.pop()
  }
}
</script>

<style scoped>
.run-queued {
  border-left: 5px solid #eed202;
}
.run-running {
  border-left: 5px solid #7cfc00;
}
.run-success {
  border-left: 5px solid #228b22;
}
.run-failed {
  border-left: 5px solid #ff0000;
}
.run-canceled {
  border-left: 5px solid #ff0000;
}
.run-wiped {
  border-left: 5px solid #3333ff;
}
.run-unknown {
  border-left: 5px solid #888888;
}
</style>

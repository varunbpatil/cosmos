<template>
  <div>
    <v-row class="mt-6">
      <v-col cols="12" md="6">
        <v-select
          :menu-props="{ offsetY: true }"
          :items="artifacts"
          item-text="name"
          item-value="id"
          label="Select artifact"
          v-model="artifactID"
          prepend-icon="mdi-file-outline"
          color="indigo"
          item-color="indigo"
        >
        </v-select>
      </v-col>
    </v-row>

    <v-card flat v-if="data" class="mt-6">
      <v-card-text style="overflow: auto">
        <pre
          class="grey--text text--darken-3"
          style="font-family: 'Roboto Mono', monospace;"
          v-html="pretty(data)"
        ></pre>
      </v-card-text>
    </v-card>

    <div v-if="error" style="white-space: pre-line" class="text-body-1 red--text text--darken-2 mt-8">{{ error }}</div>
  </div>
</template>

<script>
var ansispan = require('ansispan');
require('colors');

export default {
  data() {
    return {
      artifacts: [
        {id: 0, name: "source"},
        {id: 1, name: "destination"},
        {id: 2, name: "normalization"},
        {id: 3, name: "worker"},
        {id: 4, name: "source-config"},
        {id: 5, name: "destination-config"},
        {id: 6, name: "source-catalog"},
        {id: 7, name: "destination-catalog"},
        {id: 8, name: "before-state"},
        {id: 9, name: "after-state"},
      ],
      artifactID: 0,
      data: null,
      error: null,
      intervalID: null,
    }
  },

  methods: {
    pretty(value) {
      if (typeof value === 'string') {
        return ansispan(value)
      } else {
        return JSON.stringify(value, null, 2)
      }
    },

    fetchArtifact(artifactID) {
      artifactID = artifactID || this.artifactID

      this.$axios
        .get(`api/v1/artifacts/${this.$route.params.runID}/${artifactID}`)
        .then((response) => {
          this.data = response.data
          this.error = null
        })
        .catch((error) => {
          if (error.response) {
            this.data = null
            this.error = error.response.data.error
          }
        })
    }
  },

  watch: {
    artifactID: function(val) {
      this.fetchArtifact(val)
    }
  },

  mounted() {
    // First time artifact fetch.
    this.fetchArtifact(null)

    // Do a complete refresh every 5000ms.
    var v = this // Cannot access "this" inside setInterval.
    this.intervalID = setInterval(function() {
      v.fetchArtifact(null)
    }, 5000)
  },

  beforeDestroy() {
    clearInterval(this.intervalID)
  }
}
</script>

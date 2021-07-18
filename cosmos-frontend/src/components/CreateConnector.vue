<template>

  <v-dialog max-width="800" v-model="dialog" scrollable>
    <!-- activator button -->
    <template v-slot:activator="{ on, attrs }">
      <v-row no-gutters justify="end">
        <v-btn v-bind="attrs" v-on="on" dark tile depressed color="indigo">
          <v-icon left>mdi-plus</v-icon>
          <span>NEW</span>
        </v-btn>
      </v-row>
    </template>

    <!-- create-connector form displayed within the dialog -->
    <v-card>
      <v-toolbar flat dark dense color="indigo darken-1">
        <v-toolbar-title>Create a new {{ this.connectorType }} connector</v-toolbar-title>
        <v-spacer></v-spacer>
        <v-icon @click="dialog = false">mdi-close</v-icon>
      </v-toolbar>

      <v-card-text class="py-6">
        <v-text-field
          outlined
          color="indigo"
          label="Name"
          v-model.trim="connector.name"
          class="pt-3"
        ></v-text-field>

        <v-text-field
          outlined
          color="indigo"
          label="Docker image name"
          v-model.trim="connector.dockerImageName"
          class="pt-3"
        ></v-text-field>

        <v-text-field
          outlined
          color="indigo"
          label="Docker image tag"
          v-model.trim="connector.dockerImageTag"
          class="pt-3"
        ></v-text-field>

        <v-select
          outlined
          :menu-props="{ offsetY: true }"
          v-if="this.connectorType === 'destination'"
          label="Destination type"
          v-model.trim="connector.destinationType"
          :items="destinationTypes"
          color="indigo"
          item-color="indigo"
          class="pt-3"
        ></v-select>

        <div v-if="error" style="white-space: pre-line" class="text-body-1 red--text text--darken-2 mt-8">{{ error }}</div>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer> <!-- This moves the button to the right -->
        <v-btn tile outlined color="indigo" class="body-2 font-weight-bold" :loading="loading" @click="createConnector()">CREATE</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

</template>

<script>
export default {
  props: {
    connectorType: String
  },
  data() {
    return {
      // This connector object mirrors the connector object on the backend.
      // This is what is sent in the POST request.
      connector: {
        name: "",
        type: this.connectorType,
        dockerImageName: "",
        dockerImageTag: "",
        destinationType: "",
      },
      destinationTypes: [],
      dialog: false,
      loading: false,
      error: null
    }
  },
  watch: {
    // Clear form fields everytime the dialog opens.
    dialog: function(val) {
      if (val) {
        this.connector.name = ""
        this.connector.dockerImageName = "",
        this.connector.dockerImageTag = ""
        this.connector.destinationType = ""
        this.destinationTypes = []
        this.loading = false
        this.error = null

        this.$axios
          .get("/api/v1/connectors/destination-types")
          .then((response) => {
            this.destinationTypes = response.data
          })
      }
    }
  },
  methods: {
    createConnector() {
      // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
      this.loading = true
      this.error = null

      this.$axios
        .post("/api/v1/connectors", this.connector)
        .then(() => {
          // Close the dialog.
          this.dialog = false
          this.$emit("create", this.connector.name)
        })
        .catch((error) => {
          if (error.response) {
            /*
             * The request was made and the server responded with a
             * status code that falls out of the range of 2xx
             */
            console.log(error.response.data);
            console.log(error.response.status);
            console.log(error.response.headers);

            this.error = error.response.data.error
          } else if (error.request) {
            /*
             * The request was made but no response was received, `error.request`
             * is an instance of XMLHttpRequest in the browser and an instance
             * of http.ClientRequest in Node.js
             */
            console.log(error.request);
          } else {
            // Something happened in setting up the request and triggered an Error
            console.log('Error', error.message);
          }
          console.log(error.config);
        })
        .finally(() => {
          this.loading = false
        })
    },
  }
}
</script>

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

    <!-- create-endpoint form displayed within the dialog -->
    <v-card>
      <v-toolbar flat dark dense color="indigo darken-1">
        <v-toolbar-title>Create a new {{ this.endpointType }} endpoint</v-toolbar-title>
        <v-spacer></v-spacer>
        <v-icon @click="dialog = false">mdi-close</v-icon>
      </v-toolbar>

      <v-card-text class="py-6">
        <!--endpoint name-->
        <v-text-field
          outlined
          color="indigo"
          label="Name"
          v-model.trim="endpoint.name"
          class="pt-3"
        ></v-text-field>

        <!--selection box for connector-->
        <!--item-text prop controls what to display inside the selection box-->
        <!--item-value prop controls what value should be associated with a particular selection and what value is assigned to the v-model-->
        <v-autocomplete
          outlined
          :loading="!connectors"
          :items="connectors"
          item-text="name"
          item-value="id"
          v-model="endpoint.connectorID"
          :label="`${capitalize(endpointType)} connector`"
          color="indigo"
          item-color="indigo"
          clearable
          class="pt-3"
        ></v-autocomplete>

        <div v-if="form">
          <div v-for="(f, idx) in form.spec" :key="idx">
            <!--display all text fields in the configuration form as a text field-->
            <v-text-field
              outlined
              v-if="f.type === 'string' && !f.enum && dependencySatisfied(f, form)"
              :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
              :placeholder="f.examples ? f.examples.toString() : ''"
              :hint="f.description || ''"
              v-model.trim="f.value"
              :type="f.secret ? 'password' : ''"
              color="indigo"
              class="pt-3"
            >
              <!--This is to parse html content in hint-->
              <template v-slot:message="{message, key}">
                <div v-html="message" :key="key"></div>
              </template>
            </v-text-field>

            <!--display all number fields in the configuration form as a text field with a number rule-->
            <v-text-field
              outlined
              v-if="(f.type === 'number' || f.type === 'integer') && !f.enum && dependencySatisfied(f, form)"
              :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
              :placeholder="f.examples ? f.examples.toString() : ''"
              :hint="f.description || ''"
              v-model.number="f.value"
              :rules="rules"
              :type="f.secret ? 'password' : ''"
              color="indigo"
              class="pt-3"
            >
              <!--This is to parse html content in hint-->
              <template v-slot:message="{message, key}">
                <div v-html="message" :key="key"></div>
              </template>
            </v-text-field>

            <!--display enum and array of enum-->
            <v-select
              outlined
              v-if="f.enum && dependencySatisfied(f, form)"
              :items="f.enum"
              :menu-props="{ offsetY: true }"
              :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
              :placeholder="f.examples ? f.examples.toString() : ''"
              :hint="f.description || ''"
              v-model="f.value"
              :multiple="f.multiple"
              color="indigo"
              item-color="indigo"
              class="pt-3"
            >
              <!--This is to parse html content in hint-->
              <template v-slot:message="{message, key}">
                <div v-html="message" :key="key"></div>
              </template>
            </v-select>

            <!--we currently have no handling for non-enum arrays. i.e, arrays which take arbitrary user input-->

            <!--display all boolean fields in the configuration form as a checkbox-->
            <v-checkbox
              v-if="f.type === 'boolean' && dependencySatisfied(f, form)"
              :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
              :hint="f.description || ''"
              v-model="f.value"
              color="indigo"
              class="pb-3"
            >
              <!--This is to parse html content in hint-->
              <template v-slot:message="{message, key}">
                <div v-html="message" :key="key"></div>
              </template>
            </v-checkbox>
          </div>
        </div>
        <div v-else>
          <v-row v-if="endpoint.connectorID && !error" align-content="center" justify="center" class="mt-4">
            <v-col class="text-body-1 text-center" cols="12">Loading connection specification</v-col>
            <v-col cols="6"><v-progress-linear color="indigo" indeterminate height="4"></v-progress-linear></v-col>
          </v-row>
        </div>

        <div v-if="error" style="white-space: pre-line" class="text-body-1 red--text text--darken-2 mt-8">{{ error }}</div>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer> <!-- This moves the button to the right -->
        <v-btn tile outlined color="indigo" class="body-2 font-weight-bold" :loading="loading" :disabled="!form" @click="createEndpoint()">CREATE</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

</template>


<script>
const _ = require('lodash')

export default {

  props: {
    endpointType: String
  },

  data() {
    return {
      // This "endpoint" object mirrors the endpoint object on the backend.
      // This is what is sent via the POST request.
      endpoint: {
        name: "",
        type: this.endpointType,
        connectorID: null, // the v-autocomplete component sets this to the appropriate value (see 'item-value' prop)
        config: null, // this will be manually set to the "form" before creating endpoint
      },
      connectors: [], // this has to be a non-null value for v-autocomplete to be rendered correctly.
      form: null,
      dialog: false,
      loading: false,
      error: null,
      rules: [
        v => /^[-+]?\d+[.]?\d*$/.test(v) || 'This field only accepts numbers'
      ]
    }
  },

  watch: {

    // Clear form fields everytime the dialog opens.
    // Also, fetch all the connectors for the given endpointType.
    dialog: function(val) {
      if (val) {
        this.endpoint.name = ""
        this.endpoint.connectorID = null
        this.endpoint.config = null
        this.connectors = [] // this has to be an empty array, otherwise v-autocomplete will throw errors.
        this.form = null
        this.loading = false
        this.error = null

        this.$axios
          .get("/api/v1/connectors?type=" + this.endpointType)
          .then(response => {
            this.connectors = response.data.connectors
          })
      }
    },

    // When 'endpoint.connectorID' is set by the v-autocomplete field,
    // we need to fetch the configuration form for that connector.
    'endpoint.connectorID': function(val) {
      if (val) {
        this.$axios
          .get(`/api/v1/connectors/${this.endpoint.connectorID}/connection-spec-form`)
          .then(response => {
            this.form = response.data
          })
          .catch((error) => {
            if (error.response) {
              this.error = error.response.data.error
            }
          })
      } else {
        // Whenever the v-autocomplete field is "cleared", reset the form.
        this.form = null
        this.error = null
      }
    }
  },

  methods: {
    // Capitalize the first character of the given string.
    capitalize(s) {
      return s.charAt(0).toUpperCase() + s.slice(1)
    },

    dependencySatisfied(field, form) {
      if (field.ignore) {
        return false
      }
      if (field.dependsOnIdx === null) {
        return true
      }
      if (field.dependsOnValue.includes(form.spec[field.dependsOnIdx].value)) {
        return this.dependencySatisfied(form.spec[field.dependsOnIdx], form)
      }
      return false
    },

    createEndpoint() {
      // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
      this.loading = true
      this.error = null

      // We first make a deep copy of the "endpoint" so that it doesn't get changed from underneath us.
      let _endpoint = _.cloneDeep(this.endpoint)
      _endpoint.config = _.cloneDeep(this.form)

      this.$axios
        .post("/api/v1/endpoints", _endpoint)
        .then(() => {
          // Close the dialog.
          this.dialog = false
          this.$emit("create", _endpoint.name)
        })
        .catch((error) => {
          if (error.response) {
            this.error = error.response.data.error
          }
        })
        .finally(() => {
          this.loading = false
        })
    },
  }
}
</script>

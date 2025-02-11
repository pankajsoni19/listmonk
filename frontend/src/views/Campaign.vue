<template>
  <section class="campaign">
    <header class="columns page-header">
      <div class="column is-6">
        <p v-if="isEditing && data.status" class="tags">
          <b-tag v-if="isEditing" :class="data.status">
            {{ $t(`campaigns.status.${data.status}`) }}
          </b-tag>
          <b-tag v-if="data.type === 'optin'" :class="data.type">
            {{ $t('lists.optin') }}
          </b-tag>
          <span v-if="isEditing" class="has-text-grey-light is-size-7" :data-campaign-id="data.id">
            {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
            {{ $t('globals.fields.uuid') }}: <copy-text :text="data.uuid" />
          </span>
        </p>
        <h4 v-if="isEditing" class="title is-4">
          {{ data.name }}
        </h4>
        <h4 v-else class="title is-4">
          {{ $t('campaigns.newCampaign') }}
        </h4>
      </div>

      <div class="column is-6">
        <div v-if="$can('campaigns:manage')" class="buttons">
          <b-field grouped v-if="isEditing && (canEdit || canEditWindow)">
            <b-field expanded>
              <b-button
                expanded
                @click="() => onSubmit('update')"
                :loading="loading.campaigns"
                type="is-primary"
                icon-left="content-save-outline"
                data-cy="btn-save"
              >
                {{ $t('globals.buttons.saveChanges') }}
              </b-button>
            </b-field>
            <b-field expanded v-if="canStart">
              <b-button
                expanded
                @click="startCampaign"
                :loading="loading.campaigns"
                type="is-primary"
                icon-left="rocket-launch-outline"
                data-cy="btn-start"
              >
                {{ $t('campaigns.start') }}
              </b-button>
            </b-field>
            <b-field expanded v-if="canSchedule">
              <b-button
                expanded
                @click="startCampaign"
                :loading="loading.campaigns"
                type="is-primary"
                icon-left="clock-start"
                data-cy="btn-schedule"
              >
                {{ $t('campaigns.schedule') }}
              </b-button>
            </b-field>
            <b-field expanded v-if="canUnSchedule">
              <b-button
                expanded
                @click="unscheduleCampaign"
                :loading="loading.campaigns"
                type="is-primary"
                icon-left="clock-start"
                data-cy="btn-unschedule"
              >
                {{ $t('campaigns.unSchedule') }}
              </b-button>
            </b-field>
          </b-field>
        </div>
      </div>
    </header>

    <b-loading :active="loading.campaigns" />

    <b-tabs type="is-boxed" :animated="false" v-model="activeTab" @input="onTab">
      <b-tab-item
        :label="$tc('globals.terms.campaign')"
        label-position="on-border"
        value="campaign"
        icon="rocket-launch-outline"
      >
        <section class="wrap">
          <div class="columns">
            <div class="column is-7">
              <form @submit.prevent="() => onSubmit(isNew ? 'create' : 'update')">
                <b-field :label="$t('globals.fields.name')" label-position="on-border">
                  <b-input
                    :maxlength="200"
                    :ref="'focus'"
                    v-model="form.name"
                    name="name"
                    :disabled="!canEdit"
                    :placeholder="$t('globals.fields.name')"
                    required
                    autofocus
                  />
                </b-field>

                <b-field :label="$t('campaigns.subject')" label-position="on-border">
                  <b-input
                    :maxlength="5000"
                    v-model="form.subject"
                    name="subject"
                    :disabled="!canEdit"
                    :placeholder="$t('campaigns.subject')"
                    required
                  />
                </b-field>

                <list-selector
                  v-model="form.lists"
                  :selected="form.lists"
                  :all="lists.results"
                  :disabled="!canEdit"
                  :label="$t('globals.terms.lists')"
                  :placeholder="$t('campaigns.sendToLists')"
                />

                <b-field :label="$tc('globals.terms.template')" label-position="on-border">
                  <b-select
                    :placeholder="$tc('globals.terms.template')"
                    v-model="form.templateId"
                    name="template"
                    :disabled="!canEdit"
                    required
                  >
                    <template v-for="t in templates">
                      <option v-if="t.type === 'campaign'" :value="t.id" :key="t.id">
                        {{ t.name }}
                      </option>
                    </template>
                  </b-select>
                </b-field>

                <div class="columns">
                  <div class="column is-4">
                    <b-field label="Traffic" label-position="on-border">
                      <b-select v-model="form.trafficType" name="run_type" :disabled="!canEdit">
                        <option value="split">Split</option>
                        <option value="duplicate">Duplicate</option>
                      </b-select>
                    </b-field>
                  </div>
                </div>

                <div class="columns">
                  <div class="column is-12">
                    <b-field
                      :label="$tc('globals.terms.messenger')"
                      label-position="on-border"
                      :message="this.displayMessage"
                      style="
                        border-radius: 5px;
                        border: 1px solid hsl(0, 0%, 86%);
                        padding: 10px;
                        box-shadow: 2px 2px 0 hsl(0, 0%, 96%);
                      "
                      :disabled="!canEditWindow"
                    >
                      <div style="display: flex; flex-direction: column">
                        <div
                          v-for="(item, index) in messengers"
                          :key="index"
                          style="display: flex; flex-direction: row; margin-top: 24px"
                        >
                          <div style="min-width: 192px; align-content: center">
                            <b-checkbox
                              v-model="selectedStates[item.uuid]"
                              :disabled="!canEditWindow"
                              >{{ item.name }}</b-checkbox
                            >
                          </div>
                          <b-field
                            label="Weight"
                            label-position="on-border"
                            :hidden="form.trafficType == 'duplicate'"
                          >
                            <b-input
                              :maxlength="2"
                              :max="10"
                              :min="1"
                              placeholder="Weight..."
                              style="width: 84px"
                              v-model="itemValues[item.uuid]"
                              :disabled="!canEditWindow"
                            />
                          </b-field>

                          <b-field
                            label="Weighted From"
                            label-position="on-border"
                            :message="$t('settings.smtp.fromEmailHelp')"
                            style="margin-left: 10px"
                          >
                            <b-input
                              placeholder="From,Weight,..."
                              style="width: 256px"
                              v-model="wFrom[item.uuid]"
                              :disabled="!canEditWindow"
                            />
                          </b-field>
                        </div>
                      </div>
                    </b-field>
                  </div>
                </div>

                <div v-if="showError" style="color: red; padding-bottom: 24px">
                  Please select at least one option and provide a weight between 1-1000, also
                  provide a weighted from.
                </div>

                <b-field :label="$t('globals.terms.tags')" label-position="on-border">
                  <b-taginput
                    v-model="form.tags"
                    name="tags"
                    :disabled="!canEdit"
                    ellipsis
                    icon="tag-outline"
                    :placeholder="$t('globals.terms.tags')"
                  />
                </b-field>
                <hr />

                <div class="columns">
                  <div class="column is-4">
                    <b-field :label="$t('campaigns.runType')" label-position="on-border">
                      <b-select v-model="form.runType" name="run_type" :disabled="!canEdit">
                        <option value="list">List</option>
                        <option value="event:sub">Event Subcription</option>
                      </b-select>
                    </b-field>
                  </div>
                </div>

                <!-- sliding window -->
                <div class="columns">
                  <div class="column is-6">
                    <b-field
                      :label="$t('campaigns.slidingWindow')"
                      :message="$t('campaigns.slidingWindowHelp')"
                    >
                      <b-switch v-model="form.slidingWindow" :disabled="!canEditWindow" />
                    </b-field>
                  </div>

                  <div class="column is-4" v-if="form.slidingWindow">
                    <b-field
                      :label="$t('campaigns.slidingWindowRate')"
                      label-position="on-border"
                      :message="$t('campaigns.slidingWindowRateHelp')"
                    >
                      <b-numberinput
                        v-model="form.slidingWindowRate"
                        type="is-light"
                        controls-position="compact"
                        placeholder="25"
                        min="1"
                        max="10000000"
                        :disabled="!canEditWindow"
                      />
                    </b-field>
                  </div>

                  <div class="column is-3" v-if="form.slidingWindow">
                    <b-field
                      :label="$t('campaigns.slidingWindowDuration')"
                      label-position="on-border"
                      :message="$t('campaigns.slidingWindowDurationHelp')"
                    >
                      <b-input
                        v-model="form.slidingWindowDuration"
                        placeholder="1h"
                        :pattern="regDuration"
                        :maxlength="10"
                        :disabled="!canEditWindow"
                      />
                    </b-field>
                  </div>
                </div>

                <div class="columns">
                  <div class="column is-4">
                    <b-field :label="$t('campaigns.sendLater')" data-cy="btn-send-later">
                      <b-switch v-model="form.sendLater" :disabled="!canEdit" />
                    </b-field>
                  </div>
                  <div class="column">
                    <br />
                    <b-field
                      v-if="form.sendLater"
                      data-cy="send_at"
                      :message="form.sendAtDate ? $utils.duration(Date(), form.sendAtDate) : ''"
                    >
                      <b-datetimepicker
                        v-model="form.sendAtDate"
                        :disabled="!canEdit"
                        :placeholder="$t('campaigns.dateAndTime')"
                        icon="calendar-clock"
                        :timepicker="{ hourFormat: '24' }"
                        :datetime-formatter="formatDateTime"
                        horizontal-time-picker
                      />
                    </b-field>
                  </div>
                </div>

                <div>
                  <p class="has-text-right">
                    <a href="#" @click.prevent="onShowHeaders" data-cy="btn-headers">
                      <b-icon icon="plus" />{{ $t('settings.smtp.setCustomHeaders') }}
                    </a>
                  </p>
                  <b-field
                    v-if="form.headersStr !== '[]' || isHeadersVisible"
                    label-position="on-border"
                    :message="$t('campaigns.customHeadersHelp')"
                  >
                    <b-input
                      v-model="form.headersStr"
                      name="headers"
                      type="textarea"
                      placeholder='[{"X-Custom": "value"}, {"X-Custom2": "value"}]'
                      :disabled="!canEdit"
                    />
                  </b-field>
                </div>
                <hr />

                <div>
                  <p class="has-text-right">
                    <a href="#" @click.prevent="onShowAttribs" data-cy="btn-attribs">
                      <b-icon icon="plus" />{{ $t('settings.smtp.setCustomAttribs') }}
                    </a>
                  </p>
                  <b-field
                    v-if="form.attribsStr !== '{}' || isAttribsVisible"
                    label-position="on-border"
                    :message="$t('campaigns.customAttribsHelp')"
                  >
                    <b-input
                      v-model="form.attribsStr"
                      name="attribs"
                      type="textarea"
                      placeholder='{"X-Custom": "value", "X-Custom2": "value"}'
                      :disabled="!canEdit"
                    />
                  </b-field>
                </div>
                <hr />

                <b-field v-if="isNew">
                  <b-button
                    native-type="submit"
                    type="is-primary"
                    :loading="loading.campaigns"
                    data-cy="btn-continue"
                  >
                    {{ $t('campaigns.continue') }}
                  </b-button>
                </b-field>
              </form>
            </div>
            <div v-if="$can('campaigns:manage')" class="column is-4 is-offset-1">
              <br />
              <div class="box">
                <h3 class="title is-size-6">
                  {{ $t('campaigns.sendTest') }}
                </h3>
                <b-field :message="$t('campaigns.sendTestHelp')">
                  <b-taginput
                    v-model="form.testEmails"
                    :before-adding="$utils.validateEmail"
                    :disabled="isNew"
                    ellipsis
                    icon="email-outline"
                    :placeholder="$t('campaigns.testEmails')"
                  />
                </b-field>
                <b-field>
                  <b-button
                    @click="() => onSubmit('test')"
                    :loading="loading.campaigns"
                    :disabled="isNew"
                    type="is-primary"
                    icon-left="email-outline"
                  >
                    {{ $t('campaigns.send') }}
                  </b-button>
                </b-field>
              </div>
            </div>
          </div>
        </section> </b-tab-item
      ><!-- campaign -->

      <b-tab-item :label="$t('campaigns.content')" icon="text" :disabled="isNew" value="content">
        <editor
          v-model="form.content"
          :id="data.id"
          :title="data.name"
          :template-id="form.templateId"
          :content-type="data.contentType"
          :body="data.body"
          :disabled="!canEdit"
        />

        <div class="columns">
          <div class="column is-6">
            <p v-if="!isAttachFieldVisible" class="is-size-6 has-text-grey">
              <a href="#" @click.prevent="onShowAttachField()" data-cy="btn-attach">
                <b-icon icon="file-upload-outline" size="is-small" />
                {{ $t('campaigns.addAttachments') }}
              </a>
            </p>

            <b-field
              v-if="isAttachFieldVisible"
              :label="$t('campaigns.attachments')"
              label-position="on-border"
              expanded
              data-cy="media"
            >
              <b-taginput
                v-model="form.media"
                name="media"
                ellipsis
                icon="tag-outline"
                ref="media"
                field="filename"
                @focus="onOpenAttach"
                :disabled="!canEdit"
              />
            </b-field>
          </div>
          <div class="column has-text-right">
            <a
              href="https://listmonk.app/docs/templating/#template-expressions"
              target="_blank"
              rel="noopener noreferer"
            >
              <b-icon icon="code" /> {{ $t('campaigns.templatingRef') }}</a
            >
            <span
              v-if="canEdit && form.content.contentType !== 'plain'"
              class="is-size-6 has-text-grey ml-6"
            >
              <a v-if="form.altbody === null" href="#" @click.prevent="onAddAltBody">
                <b-icon icon="text" size="is-small" /> {{ $t('campaigns.addAltText') }}
              </a>
              <a v-else href="#" @click.prevent="$utils.confirm(null, onRemoveAltBody)">
                <b-icon icon="trash-can-outline" size="is-small" />
                {{ $t('campaigns.removeAltText') }}
              </a>
            </span>
          </div>
        </div>

        <div v-if="canEdit && form.content.contentType !== 'plain'" class="alt-body">
          <b-input
            v-if="form.altbody !== null"
            v-model="form.altbody"
            type="textarea"
            :disabled="!canEdit"
          />
        </div> </b-tab-item
      ><!-- content -->

      <b-tab-item
        :label="$t('campaigns.archive')"
        icon="newspaper-variant-outline"
        value="archive"
        :disabled="isNew"
      >
        <section class="wrap">
          <div class="columns">
            <div class="column is-4">
              <b-field
                :label="$t('campaigns.archiveEnable')"
                data-cy="btn-archive"
                :message="$t('campaigns.archiveHelp')"
              >
                <div class="columns">
                  <div class="column">
                    <b-switch
                      data-cy="btn-archive"
                      v-model="form.archive"
                      :disabled="!canArchive"
                    />
                  </div>
                  <div class="column is-12">
                    <a
                      :href="`${serverConfig.root_url}/archive/${data.uuid}`"
                      target="_blank"
                      rel="noopener noreferer"
                      :class="{ 'has-text-grey-light': !form.archive }"
                      aria-label="$t('campaigns.archive')"
                    >
                      <b-icon icon="link-variant" />
                    </a>
                  </div>
                </div>
              </b-field>
            </div>
            <div class="column is-8 has-text-right">
              <b-field v-if="!canEdit && canArchive">
                <b-button
                  @click="onUpdateCampaignArchive"
                  :loading="loading.campaigns"
                  type="is-primary"
                  icon-left="content-save-outline"
                  data-cy="btn-save"
                >
                  {{ $t('globals.buttons.saveChanges') }}
                </b-button>
              </b-field>
            </div>
          </div>

          <div class="columns">
            <div class="column is-8">
              <b-field :label="$tc('globals.terms.template')" label-position="on-border">
                <b-select
                  :placeholder="$tc('globals.terms.template')"
                  v-model="form.archiveTemplateId"
                  name="template"
                  :disabled="!canArchive || !form.archive"
                  required
                >
                  <template v-for="t in templates">
                    <option v-if="t.type === 'campaign'" :value="t.id" :key="t.id">
                      {{ t.name }}
                    </option>
                  </template>
                </b-select>
              </b-field>
            </div>

            <div class="column has-text-right">
              <a
                v-if="!this.form.archiveMetaStr || this.form.archiveMetaStr === '{}'"
                class="button is-primary"
                href="#"
                @click.prevent="onFillArchiveMeta"
                aria-label="{}"
                ><b-icon icon="code"
              /></a>
            </div>
          </div>
          <b-field>
            <b-field
              :label="$t('campaigns.archiveSlug')"
              label-position="on-border"
              :message="$t('campaigns.archiveSlugHelp')"
            >
              <b-input
                :maxlength="200"
                :ref="'focus'"
                v-model="form.archiveSlug"
                name="archive_slug"
                data-cy="archive-slug"
                :disabled="!canArchive || !form.archive"
              />
            </b-field>
          </b-field>
          <b-field
            :label="$t('campaigns.archiveMeta')"
            :message="$t('campaigns.archiveMetaHelp')"
            label-position="on-border"
          >
            <b-input
              v-model="form.archiveMetaStr"
              name="archive_meta"
              type="textarea"
              data-cy="archive-meta"
              :disabled="!canArchive || !form.archive"
              rows="20"
            />
          </b-field>
        </section> </b-tab-item
      ><!-- archive -->
    </b-tabs>

    <b-modal scroll="keep" :aria-modal="true" :active.sync="isAttachModalOpen" :width="900">
      <div class="modal-card content" style="width: auto">
        <section expanded class="modal-card-body">
          <media is-modal @selected="onAttachSelect" />
        </section>
      </div>
    </b-modal>
  </section>
</template>

<script>
import dayjs from 'dayjs';
import htmlToPlainText from 'textversionjs';
import Vue from 'vue';
import { mapState } from 'vuex';

import CopyText from '../components/CopyText.vue';
import Editor from '../components/Editor.vue';
import ListSelector from '../components/ListSelector.vue';
import Media from './Media.vue';
import { regDuration } from '../constants';

export default Vue.extend({
  components: {
    ListSelector,
    Editor,
    Media,
    CopyText,
  },

  data() {
    return {
      isNew: false,
      isEditing: false,
      isHeadersVisible: false,
      isAttribsVisible: false,
      isAttachFieldVisible: false,
      isAttachModalOpen: false,
      activeTab: 'campaign',

      // Tracks checkbox states
      selectedStates: {},
      // Tracks number input values
      itemValues: {},
      wFrom: {},
      showError: false,
      submitted: false,

      data: {},

      // IDs from ?list_id query param.
      selListIDs: [],

      // Binds form input values.
      form: {
        archiveSlug: null,
        name: '',
        subject: '',
        headersStr: '[]',
        attribsStr: '{}',
        headers: [],
        attribs: {},
        templateId: 0,
        lists: [],
        tags: [],
        sendAt: null,
        content: { contentType: 'richtext', body: '' },
        altbody: null,
        media: [],

        // Parsed Date() version of send_at from the API.
        sendAtDate: null,
        sendLater: false,
        archive: false,
        archiveMetaStr: '{}',
        archiveMeta: {},
        testEmails: [],
      },
      regDuration,
    };
  },

  methods: {
    formatDateTime(s) {
      return dayjs(s).format('YYYY-MM-DD HH:mm');
    },

    onAddAltBody() {
      this.form.altbody = htmlToPlainText(this.form.content.body);
    },

    onRemoveAltBody() {
      this.form.altbody = null;
    },

    onShowHeaders() {
      this.isHeadersVisible = !this.isHeadersVisible;
    },

    onShowAttribs() {
      this.isAttribsVisible = !this.isAttribsVisible;
    },

    onShowAttachField() {
      this.isAttachFieldVisible = true;
      this.$nextTick(() => {
        this.$refs.media.focus();
      });
    },

    onOpenAttach() {
      this.isAttachModalOpen = true;
    },

    onAttachSelect(o) {
      if (this.form.media.some((m) => m.id === o.id)) {
        return;
      }

      this.form.media.push(o);
    },

    isUnsaved() {
      return (
        this.data.body !== this.form.content.body ||
        this.data.contentType !== this.form.content.contentType
      );
    },

    onTab(tab) {
      if (tab === 'content' && window.tinymce && window.tinymce.editors.length > 0) {
        this.$nextTick(() => {
          window.tinymce.editors[0].focus();
        });
      }

      // this.$router.replace({ hash: `#${tab}` });
      window.history.replaceState({}, '', `#${tab}`);
    },

    onFillArchiveMeta() {
      const archiveStr = `{"email": "email@domain.com", "name": "${this.$t(
        'globals.fields.name'
      )}", "attribs": {}}`;
      this.form.archiveMetaStr =
        this.$utils.getPref('campaign.archiveMetaStr') ||
        JSON.stringify(JSON.parse(archiveStr), null, 4);
    },
    selectedMessengers() {
      const filtered = this.messengers
        .filter((item) => this.selectedStates[item.uuid] && this.wFrom[item.uuid].length > 0)
        .map((item) => {
          const mrow = {
            uuid: item.uuid,
            name: item.name,
            weight: parseInt(this.itemValues[item.uuid], 10) || 1,
            wfrom: this.wFrom[item.uuid],
          };

          return mrow;
        });

      return filtered;
    },
    onSubmit(typ) {
      // type -> create | update | test
      const messengers = this.selectedMessengers();

      // console.log(JSON.stringify(messengers));

      if (messengers.length === 0) {
        this.showError = true;
        return;
      }

      if (this.form.trafficType === 'split') {
        const outofbound = messengers.find((x) => !x.weight || x.weight < 1 || x.weight > 1000);
        if (outofbound) {
          this.showError = true;
          return;
        }
      }

      // Validate custom JSON headers.
      if (this.form.headersStr && this.form.headersStr !== '[]') {
        try {
          this.form.headers = JSON.parse(this.form.headersStr);
        } catch (e) {
          this.$utils.toast(e.toString(), 'is-danger');
          return;
        }
      } else {
        this.form.headers = [];
      }

      // Validate custom Attribs headers
      if (this.form.attribsStr && this.form.attribsStr !== '{}') {
        try {
          this.form.attribs = JSON.parse(this.form.attribsStr);
        } catch (e) {
          this.$utils.toast(e.toString(), 'is-danger');
          return;
        }
      } else {
        this.form.attribs = {};
      }

      // Validate archive JSON body.
      if (this.form.archive && this.form.archiveMetaStr) {
        try {
          this.form.archiveMeta = JSON.parse(this.form.archiveMetaStr);
        } catch (e) {
          this.$utils.toast(e.toString(), 'is-danger');
          return;
        }
      } else {
        this.form.archiveMeta = {};
      }

      switch (typ) {
        case 'create':
          this.createCampaign();
          break;
        case 'test':
          this.sendTest();
          break;
        default:
          if (this.data.status === 'paused') {
            this.updateCampaignPaused();
          } else {
            this.updateCampaign();
          }
          break;
      }
    },

    getCampaign(id) {
      return this.$api.getCampaign(id).then((data) => {
        this.data = data;

        // console.log('init', JSON.stringify(this.selectedStates), JSON.stringify(this.itemValues));

        JSON.parse(data.messenger).forEach((m) => {
          this.selectedStates[m.uuid] = true;
          this.itemValues[m.uuid] = m.weight;
          this.wFrom[m.uuid] = m.wfrom;
        });

        // console.log(
        //   'updated',
        //   JSON.stringify(this.selectedStates),
        //   JSON.stringify(this.itemValues)
        // );
        this.form = {
          ...this.form,
          ...data,
          headersStr: JSON.stringify(data.headers, null, 4),
          attribsStr: JSON.stringify(data.attribs, null, 4) || '{}',
          archiveMetaStr: data.archiveMeta ? JSON.stringify(data.archiveMeta, null, 4) : '{}',

          // The structure that is populated by editor input event.
          content: { contentType: data.contentType, body: data.body },
        };

        this.isAttachFieldVisible = this.form.media.length > 0;

        this.form.media = this.form.media.map((f) => {
          if (!f.id) {
            return { ...f, filename: `❌ ${f.filename}` };
          }
          return f;
        });

        if (data.sendAt !== null) {
          this.form.sendLater = true;
          this.form.sendAtDate = dayjs(data.sendAt).toDate();
        }
      });
    },

    sendTest() {
      const data = {
        id: this.data.id,
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        messenger: JSON.stringify(this.selectedMessengers()),
        type: 'regular',
        headers: this.form.headers,
        attribs: this.form.attribs,
        tags: this.form.tags,
        template_id: this.form.templateId,
        content_type: this.form.content.contentType,
        body: this.form.content.body,
        altbody: this.form.content.contentType !== 'plain' ? this.form.altbody : null,
        subscribers: this.form.testEmails,
        media: this.form.media.map((m) => m.id),
        traffic_type: this.form.trafficType || 'split',
      };

      this.$api.testCampaign(data).then(() => {
        this.$utils.toast(this.$t('campaigns.testSent'));
      });
      return false;
    },

    createCampaign() {
      const data = {
        archiveSlug: this.form.subject,
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        content_type: 'richtext',
        messenger: JSON.stringify(this.selectedMessengers()),
        type: 'regular',
        tags: this.form.tags,
        send_at: this.form.sendLater ? this.form.sendAtDate : null,
        headers: this.form.headers,
        attribs: this.form.attribs,
        template_id: this.form.templateId,
        media: this.form.media.map((m) => m.id),
        sliding_window: this.form.slidingWindow,
        sliding_window_rate: this.form.slidingWindowRate || 1,
        sliding_window_duration: this.form.slidingWindowDuration || '1h',
        run_type: this.form.runType || 'list',
        traffic_type: this.form.trafficType || 'split',
        // body: this.form.body,
      };

      this.$api.createCampaign(data).then((d) => {
        this.$router.push({ name: 'campaign', hash: '#content', params: { id: d.id } });
      });
      return false;
    },

    async updateCampaign(typ) {
      const data = {
        archive_slug: this.form.archiveSlug,
        name: this.form.name,
        subject: this.form.subject,
        lists: this.form.lists.map((l) => l.id),
        messenger: JSON.stringify(this.selectedMessengers()),
        type: 'regular',
        tags: this.form.tags,
        send_at: this.form.sendLater ? this.form.sendAtDate : null,
        headers: this.form.headers,
        attribs: this.form.attribs,
        template_id: this.form.templateId,
        content_type: this.form.content.contentType,
        body: this.form.content.body,
        altbody: this.form.content.contentType !== 'plain' ? this.form.altbody : null,
        archive: this.form.archive,
        archive_template_id: this.form.archiveTemplateId,
        archive_meta: this.form.archiveMeta,
        media: this.form.media.map((m) => m.id),
        sliding_window: this.form.slidingWindow,
        sliding_window_rate: this.form.slidingWindowRate || 1,
        sliding_window_duration: this.form.slidingWindowDuration || '1h',
        run_type: this.form.runType || 'list',
        traffic_type: this.form.trafficType || 'split',
      };

      let typMsg = 'globals.messages.updated';
      if (typ === 'start') {
        typMsg = 'campaigns.started';
      }

      // This promise is used by startCampaign to first save before starting.
      return new Promise((resolve) => {
        this.$api.updateCampaign(this.data.id, data).then((d) => {
          this.data = d;
          this.form.archiveSlug = d.archiveSlug;
          this.$utils.toast(this.$t(typMsg, { name: d.name }));
          resolve();
        });
      });
    },

    async updateCampaignPaused() {
      const data = {
        sliding_window: this.form.slidingWindow,
        sliding_window_rate: this.form.slidingWindowRate || 1,
        sliding_window_duration: this.form.slidingWindowDuration || '1h',
        messenger: JSON.stringify(this.selectedMessengers()),
      };

      const typMsg = 'globals.messages.updated';

      return new Promise((resolve) => {
        this.$api.updateCampaignPaused(this.data.id, data).then(() => {
          this.$utils.toast(this.$t(typMsg, { name: this.data.name }));
          resolve();
        });
      });
    },

    onUpdateCampaignArchive() {
      if (this.isEditing && this.canEdit) {
        return;
      }

      const data = {
        archive: this.form.archive,
        archive_template_id: this.form.archiveTemplateId,
        archive_meta: JSON.parse(this.form.archiveMetaStr),
        archive_slug: this.form.archiveSlug,
      };

      this.$api.updateCampaignArchive(this.data.id, data).then((d) => {
        this.form.archiveSlug = d.archiveSlug;
      });
    },

    // Starts or schedule a campaign.
    startCampaign() {
      if (!this.canStart && !this.canSchedule) {
        return;
      }

      this.$utils.confirm(null, () => {
        // First save the campaign.
        this.updateCampaign().then(() => {
          // Then start/schedule it.
          let status = '';
          if (this.canSchedule) {
            status = 'scheduled';
          } else if (this.canStart) {
            status = 'running';
          } else {
            return;
          }

          this.$api.changeCampaignStatus(this.data.id, status).then(() => {
            this.$router.push({ name: 'campaigns' });
          });
        });
      });
    },

    unscheduleCampaign() {
      this.$api.changeCampaignStatus(this.data.id, 'draft').then((d) => {
        this.data = d;
        this.form.archiveSlug = d.archiveSlug;
      });
    },
  },

  computed: {
    ...mapState(['serverConfig', 'loading', 'lists', 'templates']),

    canEdit() {
      return (
        this.isNew ||
        this.data.status === 'draft' ||
        this.data.status === 'scheduled' ||
        this.data.status === 'paused'
      );
    },

    canEditWindow() {
      return (
        this.isNew ||
        this.data.status === 'draft' ||
        this.data.status === 'scheduled' ||
        this.data.status === 'paused'
      );
    },

    displayMessage() {
      let str = '';
      if (this.form.trafficType === 'split') {
        str = 'For spliting data choose integer weights between 1 to 1000';
      } else {
        str = 'For duplicate all messages are replicated on all selected messengers';
      }

      str += ', additionally give weighted Sender values';

      return str;
    },
    canSchedule() {
      return this.data.status === 'draft' && this.data.sendAt;
    },

    canUnSchedule() {
      return this.data.status === 'scheduled' && this.data.sendAt;
    },

    canStart() {
      return this.data.status === 'draft' || this.data.status === 'paused';
    },

    canArchive() {
      return this.data.status !== 'cancelled' && this.data.type !== 'optin';
    },

    selectedLists() {
      if (this.selListIDs.length === 0 || !this.lists.results) {
        return [];
      }

      return this.lists.results.filter((l) => this.selListIDs.indexOf(l.id) > -1);
    },
    messengers() {
      return this.serverConfig.messengers.map((item) => {
        const row = {
          uuid: item.uuid,
          name: item.name,
          selected: false,
          weight: 1,
          wfrom: '',
        };

        return row;
      });
    },
  },

  beforeRouteLeave(to, from, next) {
    if (this.isUnsaved()) {
      this.$utils.confirm(this.$t('globals.messages.confirmDiscard'), () => next(true));
      return;
    }
    next(true);
  },

  watch: {
    selectedLists() {
      this.form.lists = this.selectedLists;
    },
  },

  mounted() {
    window.onbeforeunload = () => this.isUnsaved() || null;

    // New campaign.
    const { id } = this.$route.params;
    if (id === 'new') {
      this.isNew = true;

      if (this.$route.query.list_id) {
        // Multiple list_id query params.
        let strIds = [];
        if (typeof this.$route.query.list_id === 'object') {
          strIds = this.$route.query.list_id;
        } else {
          strIds = [this.$route.query.list_id];
        }

        this.selListIDs = strIds.map((v) => parseInt(v, 10));
      }
    } else {
      const intID = parseInt(id, 10);
      if (intID <= 0 || Number.isNaN(intID)) {
        this.$utils.toast(this.$t('campaigns.invalid'));
        return;
      }

      this.isEditing = true;
    }

    // Get templates list.
    this.$api.getTemplates().then((data) => {
      if (data.length > 0) {
        if (!this.form.templateId) {
          this.form.templateId = data.find((i) => i.isDefault === true)?.id || data[0]?.id;
        }
      }
    });

    // Fetch campaign.
    if (this.isEditing) {
      this.getCampaign(id).then(() => {
        if (this.$route.hash !== '') {
          this.activeTab = this.$route.hash.replace('#', '');
        }
      });
    }

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>

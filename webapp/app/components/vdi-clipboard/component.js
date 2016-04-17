import Ember from 'ember';
import VdiWindowComponent from 'nanocloud/components/vdi-window/component';

export default VdiWindowComponent.extend({

  remoteSession: Ember.inject.service('remote-session'),
  hasFocus: false,
  localClipboardContent: null,

  updateCloudClipboardOnTyping: function() {
    this.get('remoteSession').setCloudClipboard(this.get('connectionName'), this.get('cloudClipboardContent'));
  }.observes('cloudClipboardContent'),

  init: function() {
    this._super(...arguments);
    var connectionName = this.get('connectionName');
    Ember.defineProperty(this, 'localClipboardContent', Ember.computed.alias(`remoteSession.openedGuacSession.${connectionName}.localClipboard`));
    Ember.defineProperty(this, 'cloudClipboardContent', Ember.computed.alias(`remoteSession.openedGuacSession.${connectionName}.cloudClipboard`));
  },

  mouseEnter() {
      this.set('hasFocus', false);
      this.get('remoteSession').pauseInputs(this.get('connectionName'));
  },

  mouseLeave() {
      this.set('hasFocus', true);
      this.get('remoteSession').restoreInputs(this.get('connectionName'));
  },

  actions: {

    savePasteToCloud() {
      this.get('remoteSession').setCloudClipboard(this.get('connectionName'), this.get('localClipboardContent'));
    },

    savePasteToLocal() {
      this.get('remoteSession').setLocalClipboard(this.get('connectionName'), this.get('cloudClipboardContent'));
    },
  }
});


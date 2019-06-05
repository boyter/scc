using Toybox.Application;
using Toybox.Time;

class HassIQApp extends Application.AppBase {
	var state = new HassIQState();
	var view;
	var delegate;

	function initialize() {
		AppBase.initialize();
	}

	function onStart(state) {
		onSettingsChanged();
		self.state.load(getProperty("state"));
		self.state.setToken(getProperty("token"));
		self.state.setRefreshToken(getProperty("refresh_token"));
		var expireValue = getProperty("expire_time");
		if (expireValue != null) {
			self.state.setExpireTime(new Time.Moment(expireValue));
		}

		var selected = getProperty("selected");
		if (selected != null) {
			for (var i=0; i<self.state.entities.size(); ++i) {
				if (self.state.entities[i][:entity_id].equals(selected)) {
					self.state.selected = self.state.entities[i];
					break;
				}
			}
		}
	}

	function onStop(state) {
		setProperty("state", self.state.save());
		setProperty("token", self.state.getToken());
		setProperty("refresh_token", self.state.getRefreshToken());
		var expireTime = self.state.getExpireTime();
		if (expireTime != null) {
			setProperty("expire_time", expireTime.value());
		}

		var selected = null;
		if (self.state.selected != null) {
			selected = self.state.selected[:entity_id];
		}
		setProperty("selected", selected);

		self.state.destroy();
	}

	function onSettingsChanged() {
		var host = getProperty("host");
		var group = getProperty("group");
		var llat = getProperty("llat");
		var textsize = getProperty("textsize");

		state.setHost(host);
		state.setGroup(group);
		state.setLlat(llat);
		state.setTextsize(textsize);

		if (view != null) {
			view.requestUpdate();
		}
	}

	function getInitialView() {
		delegate = new HassIQDelegate(state);
		view = new HassIQView(state);

		return [view, delegate];
	}
}
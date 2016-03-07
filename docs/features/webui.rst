.. _webui:

WebUI
=====

Until such time that goiardi finally reaches 1.0.0 and can use the official Chef management console (free for up to 25 nodes), goiardi can use the old Chef WebUI if you're so inclined. Goiardi can use webui without any tweaks, but there's a forked webui repo at https://github.com/ctdk/chef-server-webui with some customizations to make it a little bit easier to use. There's not currently a smooth easy way to install this webui yet, unfortunately, but it's just a basic Rails app. It hasn't been merged into master yet, but there's a webui installation recipe you can look at at https://github.com/ctdk/chef-goiardi/tree/rt-work for guidance until a smoother procedure is worked out.

Chef-browser is another web based frontend for Chef, available at https://github.com/3ofcoins/chef-browser, that works with goiardi. It doesn't have all the features of chef-webui, but it's very nice on its own merits.

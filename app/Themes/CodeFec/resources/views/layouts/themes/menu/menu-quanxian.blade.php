@if (!core_menu_pd($key))
    @if(call_user_func($value['quanxian'])===true)
        @include('App::layouts.themes.menu.single')
    @endif
@else
    @if(call_user_func($value['quanxian'])===true)
        @include('App::layouts.themes.menu.multiple')
    @endif
@endif

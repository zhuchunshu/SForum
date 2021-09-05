@if (!core_menu_pd($key))
    @if(auth()->Class()['permission-value']>=$value['quanxian'])
        @include('plugins.Core.layouts.themes.menu.single')
    @endif
@else
    @if(auth()->Class()['permission-value']>=$value['quanxian'])
        @include('plugins.Core.layouts.themes.menu.multiple')
    @endif
@endif

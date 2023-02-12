<div class="order-first my-2 my-md-0 flex-grow-1 flex-md-grow-0 order-md-last d-flex flex-row">
    @foreach(Itf()->get('ui-common-header-right-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
    {{--    夜间模式--}}
    <div class="mx-2 d-md-none align-self-center">
        @if(session()->get('theme','theme-dark')!=="theme-dark")
            <a href="#" name="core_update_theme" class="px-0 nav-link">
                <!-- Download SVG icon from http://tabler-icons.io/i/bell -->
                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-moon" width="24" height="24"
                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                     stroke-linejoin="round">
                    <desc>Download more icon variants from https://tabler-icons.io/i/moon</desc>
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                    <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"></path>
                </svg>
            </a>
        @else
            <a href="#" name="core_update_theme" class="px-0 nav-link">
                <!-- Download SVG icon from http://tabler-icons.io/i/bell -->
                <svg class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                     fill="none" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M0 0h24v24H0z" stroke="none"></path>
                    <circle cx="12" cy="12" r="4"></circle>
                    <path d="M3 12h1m8-9v1m8 8h1m-9 8v1M5.6 5.6l.7.7m12.1-.7l-.7.7m0 11.4l.7.7m-12.1-.7l-.7.7"></path>
                </svg>
            </a>
        @endif
    </div>
</div>

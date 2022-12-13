<div class="row row-cards">


    @foreach(Itf()->get('ui-tags-right-start-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
    @foreach(Itf()->get('ui-common-right-start-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach

    @foreach(Itf()->get('ui-tags-right-end-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
    @foreach(Itf()->get('ui-common-right-end-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
</div>
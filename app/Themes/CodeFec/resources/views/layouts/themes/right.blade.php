<div class="order-first my-2 my-md-0 flex-grow-1 flex-md-grow-0 order-md-last">
    @foreach(Itf()->get('ui-common-header-right-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
</div>

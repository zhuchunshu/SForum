<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateUsersPmTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('users_pm', function (Blueprint $table) {
            $table->longText('post_id')->change();
	        $table->renameColumn('post_id', 'message')->change();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('users_pm', function (Blueprint $table) {
            //
        });
    }
}

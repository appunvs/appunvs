// Profile tab — account center.  No Box list here (that lives in the
// Chat header switcher).  Layout sections, top to bottom:
//
//   1. Account header (avatar / name / plan summary)
//   2. Usage quota bars (today's chat count + storage)
//   3. Settings (theme override; future: language)
//   4. Devices (logged-in surfaces; future: register-new flow)
//   5. Footer actions (sign out, manage subscription, version)
//
// All sections live inside Cards so the page reads as discrete blocks
// rather than a long edge-to-edge list.
import { ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { useTheme } from '@/theme';
import { useThemeOverrideStore, type ThemeOverride } from '@/theme';
import { Badge, Button, Card, Divider, Text } from '@/ui';
import { QuotaBar } from '@/components';

export default function ProfileTab() {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  const themeOverride = useThemeOverrideStore((s) => s.override);
  const setOverride = useThemeOverrideStore((s) => s.set);

  return (
    <ScrollView
      style={{ flex: 1, backgroundColor: theme.colors.bgPage }}
      contentContainerStyle={{
        padding: theme.spacing.l,
        paddingTop: insets.top + theme.spacing.l,
        paddingBottom: insets.bottom + theme.spacing.xxl,
        gap: theme.spacing.l,
      }}
    >
      <Text variant="h1">个人中心</Text>

      <AccountCard />

      <UsageCard />

      <SettingsCard
        themeOverride={themeOverride}
        onThemeChange={(v) => void setOverride(v)}
      />

      <DevicesCard />

      <FooterCard />
    </ScrollView>
  );
}

function AccountCard() {
  const theme = useTheme();
  return (
    <Card>
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: theme.spacing.l }}>
        <View
          style={{
            width: 56, height: 56,
            borderRadius: 28,
            backgroundColor: theme.colors.brandPale,
            alignItems: 'center', justifyContent: 'center',
          }}
        >
          <Text variant="h2" color="brandDark">u</Text>
        </View>
        <View style={{ flex: 1 }}>
          <Text variant="h3" numberOfLines={1}>未登录用户</Text>
          <Text color="textSecondary" numberOfLines={1}>guest@local</Text>
        </View>
        <Badge label="Free" tone="info" />
      </View>
      <View style={{ marginTop: theme.spacing.l, flexDirection: 'row', gap: theme.spacing.s }}>
        <Button label="登录 / 注册" variant="primary" size="sm" />
        <Button label="升级到 Pro" variant="secondary" size="sm" />
      </View>
    </Card>
  );
}

function UsageCard() {
  const theme = useTheme();
  // V1 placeholder values; the next slice wires these to GET /billing/status.
  return (
    <Card>
      <Text variant="h3" style={{ marginBottom: theme.spacing.m }}>本月用量</Text>
      <View style={{ gap: theme.spacing.m }}>
        <QuotaBar label="对话" used={0} cap={300} />
        <QuotaBar label="存储" used={0} cap={5_120} unit="MB" />
      </View>
    </Card>
  );
}

function SettingsCard({
  themeOverride,
  onThemeChange,
}: {
  themeOverride: ThemeOverride;
  onThemeChange: (next: ThemeOverride) => void;
}) {
  const theme = useTheme();
  type Choice = { id: ThemeOverride; label: string };
  const choices: Choice[] = [
    { id: null,    label: '跟随系统' },
    { id: 'light', label: '浅色' },
    { id: 'dark',  label: '深色' },
  ];
  return (
    <Card>
      <Text variant="h3">设置</Text>
      <Text variant="captionStrong" color="textSecondary" style={{ marginTop: theme.spacing.l }}>
        主题
      </Text>
      <View style={{ flexDirection: 'row', gap: theme.spacing.s, marginTop: theme.spacing.s }}>
        {choices.map((c) => (
          <Button
            key={c.id ?? 'system'}
            label={c.label}
            variant={c.id === themeOverride ? 'primary' : 'secondary'}
            size="sm"
            onPress={() => onThemeChange(c.id)}
          />
        ))}
      </View>
    </Card>
  );
}

function DevicesCard() {
  const theme = useTheme();
  return (
    <Card padding="none">
      <View style={{ padding: theme.spacing.l }}>
        <Text variant="h3">设备</Text>
      </View>
      <Divider />
      <DeviceRow name="当前设备" hint="此刻活跃" current />
      <Divider />
      <View style={{ padding: theme.spacing.l }}>
        <Button label="注册新设备" variant="secondary" size="sm" />
      </View>
    </Card>
  );
}

function DeviceRow({
  name, hint, current,
}: { name: string; hint: string; current?: boolean }) {
  const theme = useTheme();
  return (
    <View
      style={{
        paddingHorizontal: theme.spacing.l,
        paddingVertical: theme.spacing.m,
        flexDirection: 'row',
        alignItems: 'center',
        gap: theme.spacing.m,
      }}
    >
      <View style={{ flex: 1 }}>
        <Text variant="bodyStrong">{name}</Text>
        <Text variant="caption" color="textSecondary">{hint}</Text>
      </View>
      {current ? <Badge label="本机" tone="info" /> : null}
    </View>
  );
}

function FooterCard() {
  const theme = useTheme();
  return (
    <View style={{ alignItems: 'center', gap: theme.spacing.s, marginTop: theme.spacing.l }}>
      <Button label="退出登录" variant="ghost" size="sm" />
      <Text variant="caption" color="textSecondary">appunvs · v0.1 (dev)</Text>
    </View>
  );
}
